package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func hasSystemd() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

// canWritePath tests if we can create/write a file at the given path.
func canWritePath(path string) bool {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".gamepad-osk-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// runSetup performs system diagnostics (install=false) or deployment (install=true).
func runSetup(install bool) int {
	if install {
		return runInstall()
	}
	return runDiagnose()
}

func runDiagnose() int {
	fmt.Printf("gamepad-osk-no-uinput v%s system check:\n", version)
	fmt.Println()

	if os.Geteuid() == 0 {
		fmt.Println(colorYellow("  Note: running as root - results reflect root access, not your user's."))
		fmt.Println(colorDim("  Run without sudo for an accurate check of your user's permissions."))
		fmt.Println()
	}

	issues := 0
	configIssues := 0

	// 1. input access
	inputOK := false
	matches, _ := filepath.Glob("/dev/input/event*")
	if len(matches) == 0 {
		fmt.Printf("  %-14s %s no event devices found\n", "/dev/input", colorDim("[~]"))
	} else {
		for _, dev := range matches {
			if f, err := os.OpenFile(dev, os.O_RDONLY, 0); err == nil { //nolint:gosec // G304: /dev/input path
				_ = f.Close()
				inputOK = true
				break
			}
		}
		if inputOK {
			fmt.Printf("  %-14s %s readable\n", "/dev/input", colorGreen("[✓]"))
		} else {
			fmt.Printf("  %-14s %s permission denied on all event devices\n", "/dev/input", colorRed("[✗]"))
			issues++
		}
	}

	// 2. config
	userCfg := UserConfigPath()
	if _, err := os.Stat(userCfg); err == nil {
		// Suppress log output from config parsing (unknown section warnings etc.)
		log.SetOutput(io.Discard)
		cfg := LoadConfig(userCfg)
		log.SetOutput(os.Stderr)
		problems := checkConfigFile(userCfg)
		problems = append(problems, checkConfig(cfg)...)
		if len(problems) > 0 {
			fmt.Printf("  %-14s %s %s\n", "config", colorYellow("[!]"), userCfg)
			for _, issue := range problems {
				fmt.Printf("  %-14s   %s\n", "", colorYellow(issue))
			}
			fmt.Printf("  %-14s   %s\n", "", colorDim("Edit "+userCfg+" to fix"))
			configIssues++
			issues++
		} else {
			fmt.Printf("  %-14s %s %s\n", "config", colorGreen("[✓]"), userCfg)
		}
	} else {
		fmt.Printf("  %-14s %s %s (will be created on first run)\n", "config", colorDim("[~]"), userCfg)
	}

	fmt.Println()
	if issues > 0 {
		if issues > configIssues {
			fmt.Println(colorYellow("  Permission issues found. Run: sudo gamepad-osk-no-uinput --setup --install"))
		}
		if configIssues > 0 {
			fmt.Println(colorYellow("  Config issues found. Edit the config file to fix, then re-run --setup to verify."))
		}
		return 1
	}
	fmt.Println(colorGreen("  All checks passed."))
	return 0
}

func runInstall() int {
	fmt.Printf("gamepad-osk-no-uinput v%s setup:\n", version)
	fmt.Println()

	// 1. config
	userCfg := UserConfigPath()
	if _, err := os.Stat(userCfg); err == nil {
		fmt.Printf("  %-14s %s %s (already exists)\n", "config", colorDim("[~]"), userCfg)
	} else {
		dir := filepath.Dir(userCfg)
		if err := os.MkdirAll(dir, 0755); err != nil { //nolint:gosec // G301: config dir
			fmt.Printf("  %-14s %s %v\n", "config", colorRed("[✗]"), err)
		} else {
			cfg := DefaultConfig()
			f, err := os.Create(userCfg) //nolint:gosec // G304: user config path
			if err != nil {
				fmt.Printf("  %-14s %s %v\n", "config", colorRed("[✗]"), err)
			} else {
				defer func() { _ = f.Close() }()
				if err := writeINI(f, cfg); err != nil {
					fmt.Printf("  %-14s %s %v\n", "config", colorRed("[✗]"), err)
				} else {
					fmt.Printf("  %-14s %s %s\n", "config", colorGreen("[✓]"), userCfg)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(colorGreen("  Setup complete."))
	return 0
}
