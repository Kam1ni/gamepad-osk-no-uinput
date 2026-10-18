package main

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Kam1ni/gamepad-osk-no-uinput/controller"
	"github.com/Kam1ni/gamepad-osk-no-uinput/logger"
)

//nolint:revive // Match Linux evdev naming
const (
	evdevEV_KEY = 0x01
	evdevEV_ABS = 0x03
	evdevEV_SYN = 0x00

	// Standard evdev button codes
	BTN_SOUTH      = 0x130 // 304 - Xbox A, PS Cross
	BTN_EAST       = 0x131 // 305 - Xbox B, PS Circle
	BTN_NORTH      = 0x133 // 307 - Xbox Y/X (varies!)
	BTN_WEST       = 0x134 // 308 - Xbox X/Y (varies!)
	BTN_TL         = 0x136 // 310 - LB/L1
	BTN_TR         = 0x137 // 311 - RB/R1
	BTN_TL2        = 0x138 // 312 - LT/L2 (digital, Switch Pro)
	BTN_TR2        = 0x139 // 313 - RT/R2 (digital, Switch Pro)
	BTN_THUMBL     = 0x13d // 317 - L3
	BTN_THUMBR     = 0x13e // 318 - R3
	BTN_DPAD_UP    = 0x220 // 544 - DS3 D-pad (button, not axis)
	BTN_DPAD_DOWN  = 0x221 // 545
	BTN_DPAD_LEFT  = 0x222 // 546
	BTN_DPAD_RIGHT = 0x223 // 547

	ABS_X     = 0x00
	ABS_Y     = 0x01
	ABS_Z     = 0x02 // LT
	ABS_RX    = 0x03
	ABS_RY    = 0x04
	ABS_RZ    = 0x05 // RT
	ABS_HAT0X = 0x10
	ABS_HAT0Y = 0x11

	EVIOCGRAB = 0x40044590

	// EVIOCGABS(axis) = _IOR('E', 0x40 + axis, struct input_absinfo)
	// input_absinfo is 6 x int32 = 24 bytes, so _IOR size = 24
	// _IOR('E', 0x40+axis, 24) = 0x80184540 + axis
	eviocgabsBase = 0x80184540
)

// axisRange stores min/max for a single evdev axis.
type axisRange struct {
	min, max int32
}

type ActionType int

const (
	ActionNone ActionType = iota
	ActionNavigate
	ActionPress
	ActionPressStart
	ActionBackspace
	ActionSpace
	ActionEnter
	ActionShiftOn
	ActionShiftOff
	ActionCapsToggle
	ActionMouseMove
	ActionLeftClick
	ActionLeftClickRelease
	ActionRightClick
	ActionRightClickRelease
	ActionPositionToggle
	ActionPressRepeat
	ActionBackspaceRelease
	ActionSpaceRelease
	ActionEnterRelease
)

type Action struct {
	Type ActionType
	DX   int
	DY   int
}

type NavAxis struct {
	Direction int
	HeldSince time.Time
	LastMove  time.Time
}

func (n *NavAxis) RepeatInterval() time.Duration {
	if n.HeldSince.IsZero() {
		return 300 * time.Millisecond
	}
	elapsed := time.Since(n.HeldSince).Seconds()
	t := math.Min(elapsed/1.0, 1.0)
	ms := 300.0 + (80.0-300.0)*t
	return time.Duration(ms) * time.Millisecond
}

type GamepadReader struct {
	config    Config
	grabbed   bool
	navX      NavAxis
	navY      NavAxis
	dpadX     NavAxis
	dpadY     NavAxis
	mouseX    float64
	mouseY    float64
	ltActive  bool
	rtActive  bool
	actionMap ActionMap
	pressBtn  uint16 // "press" button evdev code
	shiftAxis uint16 // "shift" axis for hold behavior
	// Configurable stick axis codes
	navAxisX   uint16
	navAxisY   uint16
	mouseAxisX uint16
	mouseAxisY uint16
	deviceName string // currently unused; stored for future logging/diagnostics

	// Toggle combo state
	comboButtons    []ComboButton      // parsed from config at startup (nil = disabled)
	comboPeriodMs   int                // timing window
	btnHeld         map[uint16]bool    // current button press state
	axisState       map[uint16]float64 // current axis values (for d-pad and triggers)
	comboFirstPress time.Time          // when the first combo button was pressed
	comboFired      bool               // edge-trigger flag (prevents repeat while held)

	pendingEvents      []controller.Event
	pendingEventsMutex sync.Mutex
}

func NewGamepadReader(config Config) *GamepadReader {
	gp := &GamepadReader{
		config:    config,
		btnHeld:   make(map[uint16]bool),
		axisState: make(map[uint16]float64),
	}

	// Configure stick assignments (mouse_stick sets mouse, other stick = nav)
	mouse := config.Gamepad.MouseStick
	if mouse == "left" {
		gp.mouseAxisX, gp.mouseAxisY = ABS_X, ABS_Y
		gp.navAxisX, gp.navAxisY = ABS_RX, ABS_RY
	} else {
		gp.mouseAxisX, gp.mouseAxisY = ABS_RX, ABS_RY
		gp.navAxisX, gp.navAxisY = ABS_X, ABS_Y
	}

	// Parse toggle combo
	if config.Gamepad.ToggleCombo != "" {
		buttons, err := parseComboString(config.Gamepad.ToggleCombo)
		if err != nil {
			log.Printf("Warning: invalid toggle_combo: %v", err)
		} else {
			gp.comboButtons = buttons
			gp.comboPeriodMs = config.Gamepad.ComboPeriodMs
			if gp.comboPeriodMs <= 0 {
				gp.comboPeriodMs = 200
			}
			log.Printf("Toggle combo: %s (%dms window)", config.Gamepad.ToggleCombo, gp.comboPeriodMs)
		}
	}

	gp.initButtonMap()
	go gp.readControllerEventThread()

	return gp
}

// initButtonMap builds the action map and finds special button codes.
// Called after device open so swap_xy auto-detect can take effect.
func (gp *GamepadReader) initButtonMap() {
	gp.actionMap = BuildActionMap(gp.config.Gamepad)
	table := buttonTable(isSwapXY(gp.config.Gamepad))

	gp.pressBtn = BTN_SOUTH
	if info, ok := table[gp.config.Gamepad.Buttons.Press]; ok && !info.IsAxis {
		gp.pressBtn = info.EvdevBtn
	}
	gp.shiftAxis = ABS_Z
	if info, ok := table[gp.config.Gamepad.Buttons.Shift]; ok && info.IsAxis {
		gp.shiftAxis = info.EvdevAxis
	}
}

// isXpadDriver checks if the evdev device uses xpad, xpadneo, or xone kernel driver.
// These drivers emit BTN_X (=BTN_NORTH numerically) for the physical X button at
// the WEST position (legacy evdev convention).
func (gp *GamepadReader) isXpadDriver(devPath string) bool {
	base := filepath.Base(devPath)
	link, err := os.Readlink(filepath.Join("/sys/class/input", base, "device", "device", "driver"))
	if err != nil {
		return false
	}
	driver := filepath.Base(link)
	return driver == "xpad" || driver == "xpadneo" || driver == "xone"
}

func (gp *GamepadReader) getDeviceName(devPath string) string {
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return ""
	}
	// Find the event handler matching our device path
	eventName := devPath[strings.LastIndex(devPath, "/")+1:]
	var name string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "N: Name=") {
			name = strings.Trim(line[8:], "\"")
		} else if strings.HasPrefix(line, "H: Handlers=") && strings.Contains(line, eventName) {
			return name
		}
	}
	return ""
}

func (gp *GamepadReader) ReadEvents() []Action {
	var actions []Action
	gp.pendingEventsMutex.Lock()
	defer gp.pendingEventsMutex.Unlock()

	for _, event := range gp.pendingEvents {
		if a := gp.handleEvent(event); a.Type != ActionNone {
			actions = append(actions, a)
		}
	}
	gp.pendingEvents = []controller.Event{}

	// Adaptive repeat (stick + d-pad independently)
	now := time.Now()
	for _, pair := range []struct {
		nav *NavAxis
		isX bool
	}{
		{&gp.navX, true}, {&gp.navY, false},
		{&gp.dpadX, true}, {&gp.dpadY, false},
	} {
		if pair.nav.Direction != 0 && now.Sub(pair.nav.LastMove) >= pair.nav.RepeatInterval() {
			pair.nav.LastMove = now
			if pair.isX {
				actions = append(actions, Action{Type: ActionNavigate, DX: pair.nav.Direction})
			} else {
				actions = append(actions, Action{Type: ActionNavigate, DY: pair.nav.Direction})
			}
		}
	}

	// Mouse
	if gp.config.Mouse.Enabled && (math.Abs(gp.mouseX) > 0.01 || math.Abs(gp.mouseY) > 0.01) {
		s := float64(gp.config.Mouse.Sensitivity)
		actions = append(actions, Action{
			Type: ActionMouseMove,
			DX:   int(gp.mouseX * s),
			DY:   int(gp.mouseY * s),
		})
	}

	return actions
}

func (gp *GamepadReader) handleEvent(event controller.Event) Action {
	if event.Type != controller.EVENT_TYPE_INPUT {
		return Action{Type: ActionNone}
	}
	if event.IsAxis {
		return gp.handleAxis(event)
	}
	return gp.handleButton(event)
}

func (gp *GamepadReader) handleButton(event controller.Event) Action {
	value := event.Value
	pressed := value == 1

	// Track button state for combo detection
	gp.btnHeld[event.Target] = pressed

	// Check combo on every button event
	if toggle := gp.checkCombo(); toggle.Type != ActionNone {
		return toggle
	}

	if event.Target == gp.pressBtn {
		if pressed {
			Debugf("Button 0x%x pressed (press btn)", event.Target)
			return Action{Type: ActionPressStart}
		}
		return Action{Type: ActionPress}
	}

	// DS3 reports D-pad as buttons instead of HAT axes
	switch event.Target {
	case BTN_DPAD_UP:
		if pressed {
			return gp.updateNav(&gp.dpadY, -1, false)
		}
		return gp.updateNav(&gp.dpadY, 0, false)
	case BTN_DPAD_DOWN:
		if pressed {
			return gp.updateNav(&gp.dpadY, 1, false)
		}
		return gp.updateNav(&gp.dpadY, 0, false)
	case BTN_DPAD_LEFT:
		if pressed {
			return gp.updateNav(&gp.dpadX, -1, true)
		}
		return gp.updateNav(&gp.dpadX, 0, true)
	case BTN_DPAD_RIGHT:
		if pressed {
			return gp.updateNav(&gp.dpadX, 1, true)
		}
		return gp.updateNav(&gp.dpadX, 0, true)
	}

	if pressed {
		if action, ok := gp.actionMap.BtnPress[event.Target]; ok {
			Debugf("Button 0x%x → action %d", event.Target, action)
			return Action{Type: action}
		}
		Debugf("Button 0x%x pressed (unmapped)", event.Target)
	} else {
		if action, ok := gp.actionMap.BtnRelease[event.Target]; ok {
			return Action{Type: action}
		}
	}
	return Action{Type: ActionNone}
}

func (gp *GamepadReader) handleAxis(event controller.Event) Action {
	dz := gp.config.Gamepad.Deadzone

	// Track axis state for combo detection (d-pad and triggers)
	gp.axisState[event.Target] = event.Normalized

	// Check combo on axis events that could be combo-relevant (d-pad, triggers)
	if event.Target == ABS_HAT0X || event.Target == ABS_HAT0Y || event.Target == ABS_Z || event.Target == ABS_RZ {
		if toggle := gp.checkCombo(); toggle.Type != ActionNone {
			return toggle
		}
	}

	switch event.Target {
	case gp.navAxisX: // Nav stick X
		return gp.updateNav(&gp.navX, applyDeadzone(event.Normalized, dz), true)
	case gp.navAxisY: // Nav stick Y
		return gp.updateNav(&gp.navY, applyDeadzone(event.Normalized, dz), false)
	case ABS_HAT0X: // D-pad - separate axis to avoid stick jitter interference
		return gp.updateNav(&gp.dpadX, int(math.Round(event.Normalized)), true)
	case ABS_HAT0Y:
		return gp.updateNav(&gp.dpadY, int(math.Round(event.Normalized)), false)
	case gp.mouseAxisX: // Mouse stick X
		if math.Abs(event.Normalized) < dz {
			gp.mouseX = 0
		} else {
			gp.mouseX = event.Normalized
		}
	case gp.mouseAxisY: // Mouse stick Y
		if math.Abs(event.Normalized) < dz {
			gp.mouseY = 0
		} else {
			gp.mouseY = event.Normalized
		}
	default:
		// Check if this axis is mapped to an action (triggers)
		if action, ok := gp.actionMap.AxisActions[event.Target]; ok {
			active := event.Normalized > -0.4 // triggers: -1.0 (released) to 1.0 (full), fire at ~30%
			if event.Target == gp.shiftAxis {
				// Shift is hold-based: on/off
				if active != gp.ltActive {
					gp.ltActive = active
					if active {
						return Action{Type: ActionShiftOn}
					}
					return Action{Type: ActionShiftOff}
				}
			} else {
				// Other triggers: fire once on press (edge detection)
				if active && !gp.rtActive {
					gp.rtActive = true
					return Action{Type: action}
				} else if !active && gp.rtActive {
					gp.rtActive = false
					if releaseAction, ok := gp.actionMap.AxisRelease[event.Target]; ok {
						return Action{Type: releaseAction}
					}
				}
			}
		}
	}
	return Action{Type: ActionNone}
}

func (gp *GamepadReader) updateNav(nav *NavAxis, direction int, isX bool) Action {
	if direction != nav.Direction {
		nav.Direction = direction
		if direction != 0 {
			now := time.Now()
			nav.HeldSince = now
			nav.LastMove = now
			if isX {
				return Action{Type: ActionNavigate, DX: direction}
			}
			return Action{Type: ActionNavigate, DY: direction}
		}
		nav.HeldSince = time.Time{}
	}
	return Action{Type: ActionNone}
}

// checkCombo checks if the toggle combo is fully satisfied and returns ActionToggle if so.
// Implements edge-triggering (fires once per press) and timing window.
// Timer starts on the first button press, not when all buttons are held.
// Window expiry only takes effect on the next press cycle (requires button release to reset).
func (gp *GamepadReader) checkCombo() Action {
	if len(gp.comboButtons) == 0 {
		return Action{Type: ActionNone}
	}

	allHeld := true
	anyHeld := false
	for _, cb := range gp.comboButtons {
		if gp.isComboButtonHeld(cb) {
			anyHeld = true
		} else {
			allHeld = false
		}
	}

	if !anyHeld {
		// Full release - reset everything
		gp.comboFired = false
		gp.comboFirstPress = time.Time{}
		return Action{Type: ActionNone}
	}

	if !allHeld {
		// Partial hold - start timer on first press, reset edge trigger
		gp.comboFired = false
		if gp.comboFirstPress.IsZero() {
			gp.comboFirstPress = time.Now()
		}
		return Action{Type: ActionNone}
	}

	// All held
	if gp.comboFired {
		return Action{Type: ActionNone}
	}
	if gp.comboFirstPress.IsZero() {
		gp.comboFirstPress = time.Now()
	}
	// Window expired while buttons still held - wait for release before allowing retry
	return Action{Type: ActionNone}
}

// isComboButtonHeld checks if a single combo button is currently satisfied.
// A button is satisfied if any of its BtnCodes are held OR its axis matches.
// Analog triggers use the same 30% threshold as action detection to avoid noise.
func (gp *GamepadReader) isComboButtonHeld(cb ComboButton) bool {
	// Check digital button codes
	for _, code := range cb.BtnCodes {
		if gp.btnHeld[code] {
			return true
		}
	}
	// Check axis (d-pad or analog trigger)
	if cb.AxisCode != 0 {
		axisVal := gp.axisState[cb.AxisCode]
		if cb.AxisVal < 0 && axisVal < 0 {
			return true // d-pad up or left
		}
		if cb.AxisVal > 0 {
			// Triggers (ABS_Z, ABS_RZ): use threshold to avoid noise near zero
			// D-pad down/right (ABS_HAT0X/Y = 1): any positive value suffices
			if cb.AxisCode == ABS_Z || cb.AxisCode == ABS_RZ {
				return axisVal > -0.4
			}
			return axisVal > 0
		}
	}
	return false
}

func (gp *GamepadReader) SetSensitivity(value int) {
	gp.config.Mouse.Sensitivity = value
}

// AnyButtonHeld reports whether any digital button, d-pad direction, or
// analog trigger (over the 30% action threshold) is currently held. Used by
// the deferred-grab logic in app.go: gamepad-osk-no-uinput waits for trigger combos to
// release before calling EVIOCGRAB, so other readers of the same evdev node
// (evsieve hooks, gamepad mappers) observe the release events and don't end
// up with stuck per-button state. Mirrors evsieve's own grab=auto idiom.
func (gp *GamepadReader) AnyButtonHeld() bool {
	for _, held := range gp.btnHeld {
		if held {
			return true
		}
	}
	if gp.axisState[ABS_HAT0X] != 0 || gp.axisState[ABS_HAT0Y] != 0 {
		return true
	}
	for _, axis := range []uint16{ABS_Z, ABS_RZ} {
		norm := gp.axisState[axis]
		if norm > 0.3 {
			return true
		}
	}
	return false
}

func (gp *GamepadReader) Close() {
	controller.CloseAllDevices()
}

// isUserInGroup and logPermissionFix are in util.go

func applyDeadzone(norm, dz float64) int {
	if norm > dz {
		return 1
	} else if norm < -dz {
		return -1
	}
	return 0
}

func (gc *GamepadReader) readControllerEventThread() {
	err := controller.CreateControllers()
	if err != nil {
		logger.Err("Failed to create controllers\n%s", err.Error())
	}
	defer controller.CloseAllDevices()
	for {
		event := controller.ListenForOneEvent()
		gc.pendingEventsMutex.Lock()
		gc.pendingEvents = append(gc.pendingEvents, event)
		gc.pendingEventsMutex.Unlock()
	}
}
