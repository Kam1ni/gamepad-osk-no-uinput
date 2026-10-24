package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/bnema/libwldevices-go/virtual_keyboard"
)

const (
	uinputPath = "/dev/uinput"
	evKey      = 0x01
	evRel      = 0x02
	evSyn      = 0x00
	relX       = 0x00
	relY       = 0x01
	synReport  = 0x00
)

var kbdManager *virtual_keyboard.VirtualKeyboardManager

type Injector struct {
	keyboard               *virtual_keyboard.VirtualKeyboard
	shiftPressed           bool
	ctrlPressed            bool
	altPressed             bool
	lastSentModifiersState uint32
	pressedKeys            []int
}

func NewKbdManager() {
	if kbdManager != nil {
		return
	}
	var err error
	kbdManager, err = virtual_keyboard.NewVirtualKeyboardManager(context.Background())
	if err != nil {
		panic("Failed to create virtual keyboard manager\n" + err.Error())
	}
}

func NewInjector() (*Injector, error) {
	keyboard, err := kbdManager.CreateKeyboard()
	if err != nil {
		return nil, fmt.Errorf("Failed to create keyboard manager\n%s", err.Error())
	}
	return &Injector{keyboard: keyboard}, nil
}

func getKeyboardModifierValue(pressedModifiers []int) uint32 {
	var result uint32
	if slices.Index(pressedModifiers, 42) != -1 {
		result += 1
	}
	if slices.Index(pressedModifiers, 29) != -1 {
		result += 4
	}
	if slices.Index(pressedModifiers, 56) != -1 {
		result += 8
	}
	if slices.Index(pressedModifiers, 125) != -1 {
		result += 64
	}
	return result
}

func (inj *Injector) updateModifiers(pressedModifiers []int) {
	modifiers := getKeyboardModifierValue(pressedModifiers)
	if modifiers != inj.lastSentModifiersState {
		inj.keyboard.Modifiers(modifiers, 0, 0, 0)
		inj.lastSentModifiersState = modifiers
	}
}

func (inj *Injector) PressKey(code int, modifiers []int) {
	inj.updateModifiers(modifiers)
	inj.keyboard.TypeKey(uint32(code))
}

func (inj *Injector) TypeUnicode(str string) {
	inj.keyboard.TypeString(str)
}

func (inj *Injector) HoldKey(code int) {
	inj.keyboard.PressKey(uint32(code))
	if slices.Index(inj.pressedKeys, code) == -1 {
		inj.pressedKeys = append(inj.pressedKeys, code)
	}
}

func (inj *Injector) ReleaseKey(code int) {
	inj.keyboard.ReleaseKey(uint32(code))
	for i := len(inj.pressedKeys) - 1; i >= 0; i-- {
		inj.pressedKeys = append(inj.pressedKeys[:i], inj.pressedKeys[i+1:]...)
	}
}

func (inj *Injector) ClickMouse(button uint16, pressed bool) {
}

func (inj *Injector) MoveMouse(dx, dy int) {
}

func (inj *Injector) Close() {
	inj.keyboard.Modifiers(0, 0, 0, 0)
	for _, key := range inj.pressedKeys {
		inj.keyboard.ReleaseKey(uint32(key))
	}
	inj.keyboard.Close()
}
