package controller

import (
	"github.com/Kam1ni/gamepad-osk-no-uinput/logger"
	"sync"
)

var openedControllers = []*grabbedController{}
var openedControllerMutex = sync.Mutex{}

func addController(controller *grabbedController) {
	openedControllerMutex.Lock()
	defer openedControllerMutex.Unlock()
	openedControllers = append(openedControllers, controller)
}

func removeController(controller *grabbedController) {
	openedControllerMutex.Lock()
	defer openedControllerMutex.Unlock()
	for i := len(openedControllers) - 1; i >= 0; i-- {
		if openedControllers[i] == controller {
			openedControllers = append(openedControllers[:i], openedControllers[i+1:]...)
		}
	}
	controller.device.Close()
}

func canControllerBeGrabbed(path string) bool {
	openedControllerMutex.Lock()
	defer openedControllerMutex.Unlock()
	for _, c := range openedControllers {
		if c.device.Path() == path {
			return false
		}
	}
	return true
}

func CloseAllDevices() {
	logger.Out("Closing all devices")
	for _, c := range openedControllers {
		removeController(c)
	}
}
