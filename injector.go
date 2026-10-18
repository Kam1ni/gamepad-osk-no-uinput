package main

import (
	"time"
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

type Injector struct {
}

func NewInjector() (*Injector, error) {
	return &Injector{}, nil
}

const keyHoldDuration = 12 * time.Millisecond

func (inj *Injector) PressKey(code int, modifiers []int) {

}

func (inj *Injector) TypeUnicode(codepoint int) {
}

func (inj *Injector) HoldKey(code int) {
}

func (inj *Injector) ReleaseKey(code int) {

}

func (inj *Injector) ClickMouse(button uint16, pressed bool) {
}

func (inj *Injector) MoveMouse(dx, dy int) {
}

func (inj *Injector) Close() {

}
