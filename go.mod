module github.com/Kam1ni/gamepad-osk-no-uinput

go 1.27

require (
	github.com/bnema/libwldevices-go v0.3.0
	github.com/holoplot/go-evdev v0.0.0-20260909133534-d873ed210bda
)

require (
	github.com/bnema/wlturbo v0.3.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/holoplot/go-evdev => github.com/Kam1ni/go-evdev v0.0.0-20261014174704-b493856ae39b
