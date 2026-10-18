package controller

import (
	"errors"
	"github.com/Kam1ni/gamepad-osk-no-uinput/logger"
	"os"
	"path"
	"regexp"
)

var _JS_DEVICE_NAME_REGEX = regexp.MustCompile(`^js\d+$`)
var _EVENT_DEVICE_NAME_REGEX = regexp.MustCompile(`^event\d+$`)

func CreateControllers() error {
	logger.Out("Creating controllers")
	controllers, err := listControllers()
	if err != nil {
		return err
	}

	for _, controller := range controllers {
		eventPath, err := getInputDevicePathFromJsDevice(controller)
		if err != nil {
			logger.Err("Couldn't detect controller %s\n%s", controller, err.Error())
			continue
		}
		if !canControllerBeGrabbed(eventPath) {
			continue
		}
		controllerClone, err := grabController(eventPath)
		if err != nil {
			if err.Error() != "Cannot clone a clone" {
				logger.Err("failed to clone controller\n%s", err.Error())
			}
			continue
		}
		addController(controllerClone)
	}
	return nil
}

func listControllers() ([]string, error) {
	items, err := os.ReadDir("/dev/input")
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, item := range items {
		if !_JS_DEVICE_NAME_REGEX.MatchString(item.Name()) {
			continue
		}
		result = append(result, item.Name())
	}
	return result, nil
}

func getInputDevicePathFromJsDevice(device string) (string, error) {
	pth := path.Join("/sys/class/input", device, "device")
	items, err := os.ReadDir(pth)
	if err != nil {
		return "", err
	}

	for _, item := range items {
		if _EVENT_DEVICE_NAME_REGEX.MatchString(item.Name()) {
			return path.Join("/dev/input", item.Name()), nil
		}
	}

	return "", errors.New("Device not found")
}
