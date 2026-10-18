package main //nolint:revive // main package needs no doc comment

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"
)

var themeOrder []string

func init() {
	for name := range Themes {
		themeOrder = append(themeOrder, name)
	}
	sort.Strings(themeOrder)
}

type App struct {
	config   Config
	running  bool
	themeIdx int
	posTop   bool // true = keyboard at top of screen
	window   *Window
	mon      MonitorRect // raw monitor bounds
	winH     int32
	winX     int32 // stored for re-apply after show
	winY     int32
	margin   int32
	width    int32
	height   int32

	// Key repeat state
	repeatAction  ActionType // action to repeat (ActionNone = inactive)
	repeatStart   time.Time  // when key was first pressed
	repeatLast    time.Time  // when last repeat fired
	repeatInitial bool       // true = still in initial delay phase
	kb            *KeyboardState
	rend          *Renderer
	inj           *Injector
}

func NewApp(config Config) *App {
	return &App{
		config: config,
	}
}

func (app *App) Run() error {
	// SDL/OpenGL requires all calls from the same OS thread.
	// go-sdl2 did this internally in sdl.Init(); we must do it explicitly.
	runtime.LockOSThread()

	cfg := app.config
	ValidateConfig(&cfg)
	layout := LayoutQWERTY
	theme := GetTheme(cfg.Theme.Name)

	// Build dynamic glyph map from button config
	KeyGlyphs = BuildKeyGlyphs(cfg.Gamepad)

	// Prefer native Wayland over XWayland when running in a Wayland session
	if os.Getenv("SDL_VIDEODRIVER") == "" {
		if err := os.Setenv("SDL_VIDEODRIVER", "wayland"); err != nil {
			log.Printf("Warning: cannot set SDL_VIDEODRIVER: %v", err)
		}
	}

	// Allow the screensaver and sleep timers to fire. SDL_Init's video
	// subsystem disables the screensaver by default; on X11 that schedules
	// XResetScreenSaver every ~30s, which resets the idle counter and blocks
	// screen blank, DPMS, and suspend. The OSK has no use for an always-on
	// display, so opt out before init. On Wayland this avoids requesting
	// idle-inhibit.
	SDL3SetHint("SDL_VIDEO_ALLOW_SCREENSAVER", "1")

	if err := SDL3Init(SDL_INIT_VIDEO); err != nil {
		return err
	}
	defer SDL3Quit()

	// Get primary monitor
	mon := GetPrimaryMonitor()
	unit := CalcUnitSize(layout, mon.W, cfg)
	pad := int32(cfg.Keys.Padding) //nolint:gosec // G115: padding fits in int32
	statusH := max32(20, int32(float64(unit)*0.4))
	app.width, app.height = CalcWindowSize(layout, unit, pad, statusH)
	Debugf("Monitor: %dx%d+%d+%d, scale=%d%%, unit=%d, window=%dx%d",
		mon.W, mon.H, mon.X, mon.Y, cfg.Keys.Scale, unit, app.width, app.height)

	// Set Wayland app_id so compositor rules can target this window
	SDL3SetHint("SDL_APP_ID", "gamepad-osk-no-uinput")
	if err := os.Setenv("SDL_APP_ID", "gamepad-osk-no-uinput"); err != nil {
		log.Printf("Warning: cannot set SDL_APP_ID: %v", err)
	}

	var window *Window
	var err error
	// Wayland: create roleless surface, attach layer-shell asynchronously
	setTargetMonitor(mon.X, mon.Y)
	window, err = createWaylandWindow("gamepad-osk-no-uinput", app.width, app.height)
	if err != nil {
		return err
	}
	defer func() {
		cleanupLayerShell()
		SDL3DestroyWindow(window)
	}()

	// Attach layer-shell asynchronously -- configure arrives via SDL event pump.
	// Skip in daemon mode (starts hidden, first toggle-show will attach).
	attachLayerShellAsync(window, app.width, app.height, app.posTop, 0, cfg.Window.PanelAvoid)

	renderer, err := SDL3CreateRenderer(window)
	if err != nil {
		return err
	}
	defer SDL3DestroyRenderer(renderer)
	SDL3SetRenderVSync(renderer, 1)
	SDL3SetRenderDrawBlendMode(renderer, 1) // SDL_BLENDMODE_BLEND
	Debugf("Renderer: %s", SDL3GetRendererName(renderer))

	// Set opacity after renderer creation (SDL3 requires renderer to exist first)
	if cfg.Window.Opacity < 1.0 {
		SDL3SetWindowOpacity(window, float32(cfg.Window.Opacity))
	}

	app.rend, err = NewRenderer(renderer, theme, unit, pad)
	if err != nil {
		return err
	}
	defer app.rend.Destroy()

	app.inj, err = NewInjector()
	if err != nil {
		log.Print(colorRed("Warning: key injection disabled - " + err.Error()))
		logPermissionFix()
		log.Printf("The on-screen keyboard will display but cannot send keystrokes.")
	}
	defer func() {
		if app.inj != nil {
			app.inj.Close()
		}
	}()

	gamepad := NewGamepadReader(cfg)
	defer gamepad.Close()
	// Rebuild glyphs after gamepad open (swap_xy may have been auto-detected)
	KeyGlyphs = BuildKeyGlyphs(gamepad.config.Gamepad)

	// Init IPC
	ipc := NewIPCServer(func(cmd string) {
		switch cmd {
		case "close":
			// Todo: close running instance
		case "ping":
			// Instance check - connection success is the answer
		}
	})
	if err := ipc.Start(); err != nil {
		log.Printf("Warning: IPC server failed: %v", err)
	}
	defer ipc.Stop()

	app.kb = NewKeyboardState(layout)

	// Find current theme index for cycling
	for i, name := range themeOrder {
		if name == cfg.Theme.Name {
			app.themeIdx = i
			break
		}
	}
	app.kb.OnThemeCycle = func() {
		app.themeIdx = (app.themeIdx + 1) % len(themeOrder)
		name := themeOrder[app.themeIdx]
		app.rend.SetTheme(Themes[name])
		SaveTheme(name)
	}
	app.kb.OnThemeCycleReverse = func() {
		app.themeIdx = (app.themeIdx - 1 + len(themeOrder)) % len(themeOrder)
		name := themeOrder[app.themeIdx]
		app.rend.SetTheme(Themes[name])
		SaveTheme(name)
	}
	app.kb.OnSensitivityUp = func() {
		cfg.Mouse.Sensitivity = min(50, cfg.Mouse.Sensitivity+2)
		gamepad.SetSensitivity(cfg.Mouse.Sensitivity)
		app.rend.FlashGlyph("\u27F9", strconv.Itoa(cfg.Mouse.Sensitivity))
		SaveSensitivity(cfg.Mouse.Sensitivity)
	}
	app.kb.OnSensitivityDown = func() {
		cfg.Mouse.Sensitivity = max(1, cfg.Mouse.Sensitivity-2)
		gamepad.SetSensitivity(cfg.Mouse.Sensitivity)
		app.rend.FlashGlyph("\u27FE", strconv.Itoa(cfg.Mouse.Sensitivity))
		SaveSensitivity(cfg.Mouse.Sensitivity)
	}

	app.running = true

	for app.running {
		// Process SDL3 window events
		for evType, ok := SDL3PollEvent(); ok; evType, ok = SDL3PollEvent() {
			if evType == SDL_EVENT_QUIT {
				app.running = false
			}
		}

		// Process gamepad events (evdev - works regardless of window focus)
		for _, action := range gamepad.ReadEvents() {
			app.handleAction(action, app.kb)
		}

		// Long-press check
		if app.kb.CheckLongPress(cfg.Gamepad.LongPressMs) {
			app.rend.MarkDirty()
		}

		// Alt+Tab timeout (auto-release after 3s idle)
		if app.kb.CheckAltTabTimeout(app.inj) {
			app.rend.MarkDirty()
		}

		// Key repeat check
		if app.repeatAction != ActionNone {
			now := time.Now()
			elapsed := now.Sub(app.repeatStart).Milliseconds()
			if app.repeatInitial {
				if elapsed >= int64(cfg.Keys.RepeatDelayMs) {
					app.handleAction(Action{Type: app.repeatAction}, app.kb)
					app.repeatLast = now
					app.repeatInitial = false
				}
			} else if now.Sub(app.repeatLast).Milliseconds() >= int64(cfg.Keys.RepeatRateMs) {
				app.handleAction(Action{Type: app.repeatAction}, app.kb)
				app.repeatLast = now
			}
		}

		if app.rend.flashText != "" && time.Now().After(app.rend.flashEnd) {
			app.rend.MarkDirty()
			app.rend.flashText = ""
			app.rend.flashGlyphText = ""
		}
		if app.kb.FlashCode != 0 && time.Now().After(app.kb.FlashUntil) {
			app.rend.MarkDirty()
			app.kb.FlashCode = 0
		}

		app.rend.Draw(app.kb)

		// Frame pacing: vsync in app.rend.Draw handles active rendering.
		// Sleep when hidden, when visible but idle (nothing to draw), or
		// when stick is active (sub-vsync polling for smooth cursor).
		if app.rend.dirtyFrames <= 0 {
			pollMs := 16 // hidden idle: ~60Hz for IPC + gamepad checks
			if app.repeatAction != ActionNone {
				var nextMs int64
				if app.repeatInitial {
					nextMs = int64(cfg.Keys.RepeatDelayMs) - time.Since(app.repeatStart).Milliseconds()
				} else {
					nextMs = int64(cfg.Keys.RepeatRateMs) - time.Since(app.repeatLast).Milliseconds()
				}
				if nextMs > 0 && int(nextMs) < pollMs {
					pollMs = int(nextMs)
				}
			}
			if pollMs > 0 {
				time.Sleep(time.Duration(pollMs) * time.Millisecond)
			}
		}
	}
	return nil
}

func (app *App) handleAction(a Action, kb *KeyboardState) {
	inj := app.inj
	rend := app.rend
	rend.MarkDirty()

	switch a.Type { //nolint:exhaustive // ActionNone is filtered by caller

	case ActionNavigate:
		kb.ReleaseAltTab(inj)
		kb.Navigate(a.DX, a.DY)
		app.stopRepeat()
	case ActionPress:
		// A button released
		if app.repeatAction != ActionNone {
			// Was repeating - just stop, don't fire again
			app.stopRepeat()
		} else {
			// Accent popup select, vowel short-press, or modifier - fire on release
			kb.PressCurrent(inj)
		}
		kb.CancelLongPress()
	case ActionPressStart:
		key := kb.CurrentKey()
		if len(key.Accents) > 0 && kb.ShiftActive {
			// Shift(LT)+hold on vowel - accent popup
			kb.StartLongPress()
		} else if !key.IsModifier && key.Label != "Cfg" && key.Label != "Paste" &&
			len(key.Combo) == 0 && key.ShiftCode == 0 && key.Label != "Esc" {
			// Repeatable key - fire immediately and start repeat
			kb.PressCurrent(inj)
			app.startRepeat(ActionPressRepeat)
		}
		// Non-repeatable keys (shortcuts, Esc, Cfg, Paste, modifiers) fire on release via ActionPress
	case ActionPressRepeat:
		kb.PressCurrent(inj)
	case ActionBackspace:
		if inj != nil {
			inj.PressKey(KEY_BACKSPACE, nil)
		}
		kb.FlashKey(KEY_BACKSPACE)
		app.startRepeat(ActionBackspace)
	case ActionBackspaceRelease:
		app.stopRepeat()
	case ActionSpace:
		if inj != nil {
			inj.PressKey(KEY_SPACE, nil)
		}
		kb.FlashKey(KEY_SPACE)
		app.startRepeat(ActionSpace)
	case ActionSpaceRelease:
		app.stopRepeat()
	case ActionEnter:
		if inj != nil {
			inj.PressKey(KEY_ENTER, nil)
		}
		kb.FlashKey(KEY_ENTER)
		app.startRepeat(ActionEnter)
	case ActionEnterRelease:
		app.stopRepeat()
	case ActionShiftOn:
		kb.ShiftActive = true
		kb.ShiftHeld = true
	case ActionShiftOff:
		kb.ShiftActive = false
		kb.ShiftHeld = false
	case ActionCapsToggle:
		kb.CapsActive = !kb.CapsActive
	case ActionMouseMove:
		if inj != nil {
			inj.MoveMouse(a.DX, a.DY)
		}
	case ActionLeftClick:
		if inj != nil {
			inj.ClickMouse(0x110, true) // BTN_LEFT press
		}
	case ActionLeftClickRelease:
		if inj != nil {
			inj.ClickMouse(0x110, false) // BTN_LEFT release
		}
	case ActionRightClick:
		if inj != nil {
			inj.ClickMouse(0x111, true) // BTN_RIGHT press
		}
	case ActionRightClickRelease:
		if inj != nil {
			inj.ClickMouse(0x111, false) // BTN_RIGHT release
		}
	case ActionPositionToggle:
		app.posTop = !app.posTop
		if app.posTop {
			rend.Flash("↑ TOP")
		} else {
			rend.Flash("↓ BOTTOM")
		}
		if hasLayerShell() {
			repositionLayerSurface(app.posTop, app.margin, app.window)
		} else {
			SDL3SetWindowPosition(app.window, app.winX, app.winY)
		}
		SavePosition(app.posTop)
	}
}

func (app *App) startRepeat(action ActionType) {
	// Don't restart repeat if already repeating this action (called from repeat loop)
	if app.repeatAction == action {
		return
	}
	app.repeatAction = action
	app.repeatStart = time.Now()
	app.repeatInitial = true
}

func (app *App) stopRepeat() {
	app.repeatAction = ActionNone
}

func (app *App) show() {
	fmt.Println("Showing")
	attachLayerShellAsync(app.window, app.width, app.height, app.posTop, 0, app.config.Window.PanelAvoid)
	app.rend.MarkDirty()
	if app.config.Window.Opacity < 1.0 {
		SDL3SetWindowOpacity(app.window, float32(app.config.Window.Opacity)) // re-apply after show
	}
}

func (app *App) hide() {
	fmt.Println("Hiding")
	app.stopRepeat()
	app.kb.CapsActive = false
	app.kb.ShiftActive = false
	app.kb.ShiftHeld = false
	app.kb.CtrlActive = false
	app.kb.AltActive = false
	app.kb.MetaActive = false
	app.kb.ReleaseAltTab(app.inj)
	if app.inj != nil {
		app.inj.ReleaseKey(KEY_LEFTSHIFT)
		app.inj.ReleaseKey(KEY_LEFTCTRL)
		app.inj.ReleaseKey(KEY_LEFTALT)
		app.inj.ReleaseKey(KEY_LEFTMETA)
		app.inj.ClickMouse(0x110, false) // BTN_LEFT release
		app.inj.ClickMouse(0x111, false) // BTN_RIGHT release
	}
	destroyLayerSurface()

}
