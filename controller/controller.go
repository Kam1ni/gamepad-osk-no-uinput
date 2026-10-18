package controller

import (
	"fmt"
	"github.com/Kam1ni/gamepad-osk-no-uinput/logger"
	"strings"

	evdev "github.com/holoplot/go-evdev"
)

type grabbedController struct {
	closed     bool
	deviceName string
	device     *evdev.InputDevice
	absmax     [64]int32
	absmin     [64]int32
	absfuzz    [64]int32
	absflat    [64]int32
}

func grabController(devicePath string) (*grabbedController, error) {
	done := false
	result := &grabbedController{}
	originalDevice, err := evdev.Open(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open device\n%s", err.Error())
	}
	defer func() {
		if !done {
			originalDevice.Close()
		}
	}()
	result.device = originalDevice

	err = originalDevice.Grab()
	if err != nil {
		return nil, fmt.Errorf("failed to grab device\n%s", err.Error())
	}

	capabilities := map[evdev.EvType][]evdev.EvCode{}
	for _, t := range originalDevice.CapableTypes() {
		if t == 0x15 { // Ignore force feedback
			continue
		}
		events := originalDevice.CapableEvents(t)
		for _, e := range events {
			capabilities[t] = append(capabilities[t], e)
		}
	}

	absInfo, err := originalDevice.AbsInfos()
	if err != nil {
		return nil, fmt.Errorf("failed to get AbsInfo\n%s", err.Error())
	}
	for key, val := range absInfo {
		result.absmax[key] = val.Maximum
		result.absmin[key] = val.Minimum
		result.absflat[key] = val.Flat
		result.absfuzz[key] = val.Fuzz
	}

	controllerName, err := originalDevice.Name()
	if err != nil {
		controllerName = strings.ReplaceAll(originalDevice.Path(), "/", "_")
	}
	result.deviceName = controllerName
	done = true

	go result.run()
	return result, nil
}

func (c *grabbedController) normalizeAxis(axis int32, value int32) float64 {
	min := c.absmin[axis]
	max := c.absmax[axis]
	if max <= min {
		return 0
	}
	return (float64(value-min)/float64(max-min))*2.0 - 1.0

}

func (c *grabbedController) run() {
	defer func() {
		logger.Out("Device %s closing", c.deviceName)
		c.close()
	}()

	for {
		event, err := c.device.ReadOne()
		if err != nil {
			logger.Err("error reading event from %s\n%s", c.deviceName, err.Error())
			return
		}

		if event.Type == evdev.EV_ABS {
			sendEvent(Event{
				EVENT_TYPE_INPUT,
				uint16(event.Code),
				event.Value,
				c.normalizeAxis(int32(event.Code), event.Value),
				true,
			})
		} else if event.Type == evdev.EV_KEY {
			sendEvent(Event{
				EVENT_TYPE_INPUT,
				uint16(event.Code),
				event.Value,
				float64(event.Value),
				false,
			})
		}

	}
}

func (c *grabbedController) close() {
	if c.closed {
		return
	}
	c.closed = true
	if c.device != nil {
		c.device.Close()
	}
	removeController(c)
}
