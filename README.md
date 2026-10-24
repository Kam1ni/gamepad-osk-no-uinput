# gamepad-osk-no-uinput

Fork of github.com/0x90shell/gamepad-osk made adjusted to remove the dependency on uinput at the cost of only working in wayland.
Gamepad-controlled on-screen keyboard for Linux. Designed for couch gaming, Sunshine/Moonlight streaming, and any setup where you need to type with a controller.

No Steam dependency. Works Wayland.

![gamepad-osk-no-uinput](assets/matrix-hero.png)

## Table of Contents

- [Features](#features)
- [Controls](#controls)
- [Usage](#usage)
- [Installation](#installation)
  - [AUR (Arch Linux)](#aur-arch-linux)
  - [Pre-built binary (x86_64)](#pre-built-binary-x86_64)
  - [From source](#from-source)
  - [Bazzite / Immutable Fedora](#bazzite--immutable-fedora)
  - [Promptfont](#promptfont)
- [Configuration](#configuration)
- [Themes](#themes)
- [Show & Hide](#show--hide)
- [Wayland](#wayland)
- [License](#license)

## Features

- Full QWERTY keyboard with shortcuts row (Undo, Redo, Cut, Select All, Alt+Tab, etc.)
- Native Wayland overlay via wlr-layer-shell (Sway, Hyprland, KDE, COSMIC - no compositor rules needed)
- Evdev gamepad input (works with any controller)
- Xbox pad auto-detection (swap_xy for xpad/xpadneo/xone drivers)
- 60 color themes (cycle live with Cfg key, or set via config/flag)
- Promptfont controller-agnostic button glyphs on mapped keys
- Mouse cursor via right stick + R3/RB click (hold to drag)
- Live mouse sensitivity adjustment (Shift + arrow keys, saved to config)
- Auto-reconnect on controller disconnect (timeout, power-off, unplug)
- Key repeat on hold (configurable delay and rate)
- Shift layer, Caps Lock, accent popup (Shift + hold on vowels)
- Paste/Copy key, media keys (Play/Pause, Mute)
- Always-on-top, no focus stealing (types into whatever app has focus)
- Multi-monitor aware (positions on primary monitor)
- Configurable: buttons, sticks, theme, scale, position, opacity, deadzone
- Single static binary, ~5MB

## Controls

| Input | Action |
|-------|--------|
| Left stick / D-pad | Navigate keyboard |
| Right stick | Move mouse cursor |
| A | Press highlighted key (hold to repeat) |
| B | Close keyboard |
| X | Backspace (hold to repeat) |
| Y | Space (hold to repeat) |
| LT (hold) | Shift |
| RT | Enter (hold to repeat) |
| RB | Left mouse click (hold to drag) |
| LB | Right mouse click |
| Mouse stick click (R3) | Left mouse click |
| Nav stick click (L3) | Caps Lock |
| Start | Toggle keyboard top/bottom |
| Shift (LT) + hold A (on vowel) | Accent popup (é, ñ, ü, etc.) |
| Shift (LT) + up/down arrow | Adjust mouse sensitivity (saved to config) |
| Cfg key | Cycle themes (Shift+Cfg = reverse) |
| Toggle combo (configurable) | Show/hide keyboard |

## Usage

```
gamepad-osk-no-uinput                          # start (auto-detect gamepad)
gamepad-osk-no-uinput --device /dev/input/X    # use specific device
gamepad-osk-no-uinput --theme synthwave        # start with theme
gamepad-osk-no-uinput --config /path/to/config  # use specific config file
gamepad-osk-no-uinput --daemon                 # start hidden, B hides instead of exiting
gamepad-osk-no-uinput --help                   # show all options
```

## Installation

**Runtime dependencies:** SDL3, SDL3_ttf, wayland, [promptfont](https://codeberg.org/shinmera/promptfont)

**Build dependencies:** Go, SDL3 (dev), SDL3_ttf (dev), wayland (dev), wayland-protocols (dev)

Package names vary by distro. Find yours below:

| Distro | Base | Instructions |
|--------|------|--------------|
| Manjaro, EndeavourOS | Arch | [AUR / Arch](#aur-arch-linux) |
| CachyOS, Garuda, ChimeraOS | Arch | [AUR / Arch](#aur-arch-linux) |
| Nobara | Fedora | [Fedora](#from-source) |
| Bazzite, Bluefin | Fedora Atomic | [Immutable Fedora](#bazzite--immutable-fedora) |
| Pop!_OS, Linux Mint | Ubuntu | [Debian / Ubuntu](#from-source) |

### AUR (Arch Linux)

The fastest way to install on Arch:

```bash
yay -S gamepad-osk-no-uinput-bin   # pre-built binary from GitHub release
yay -S gamepad-osk-no-uinput-git   # build from latest source
```

AUR packages install all dependencies, and promptfont automatically.

To auto-update `-git` packages when upstream changes, enable devel checking:

```bash
yay --devel --save
```

### Pre-built binary (x86_64)

Download the binary and support files from the [releases page](https://github.com/0x90shell/gamepad-osk-no-uinput/releases). This is the fastest option for non-Arch distros.

Install runtime dependencies for your distro first (see [Promptfont](#promptfont) and the dep commands in [From source](#from-source)), then:

```bash
chmod +x gamepad-osk-no-uinput
sudo install -Dm755 gamepad-osk-no-uinput /usr/bin/gamepad-osk-no-uinput
sudo udevadm control --reload-rules
```

A default config is auto-copied to `~/.config/gamepad-osk-no-uinput/` on first run.

### From source

Clone and build:

```bash
git clone https://github.com/0x90shell/gamepad-osk-no-uinput.git
cd gamepad-osk-no-uinput
go build -o gamepad-osk-no-uinput .
sudo install -Dm755 gamepad-osk-no-uinput /usr/bin/gamepad-osk-no-uinput
sudo install -Dm644 config.example /usr/share/gamepad-osk-no-uinput/config
sudo udevadm control --reload-rules
```

Install build dependencies for your distro before building:

**Arch:**

```bash
sudo pacman -S go sdl3 sdl3_ttf wayland wlr-protocols
yay -S ttf-promptfont
```

**Fedora / Nobara:**

```bash
sudo dnf install golang SDL3-devel SDL3_ttf-devel wayland-devel wayland-protocols-devel
```

**Debian 13+ / Ubuntu 25.04+:**

```bash
sudo apt install golang-go libsdl3-dev libsdl3-ttf-dev libwayland-dev wayland-protocols
```

SDL3 is not available on Ubuntu 24.04 or Debian 12. Use a newer release or build SDL3 from source.

### Bazzite / Immutable Fedora

gamepad-osk-no-uinput needs raw access to `/dev/input` and `/dev/uinput` for gamepad reading and key injection. This rules out Flatpak. On immutable Fedora-based systems (Bazzite, Bluefin, Fedora Atomic), layer the runtime dependencies and reboot:

```bash
rpm-ostree install SDL3 SDL3_ttf
systemctl reboot
```

The udev rule goes in `/etc` which is writable:

```bash
sudo cp gamepad-osk-no-uinput.udev /etc/udev/rules.d/80-gamepad-osk-no-uinput.rules
sudo udevadm control --reload-rules
```

**Option 1: Pre-built binary (recommended).** Download from the [releases page](https://github.com/0x90shell/gamepad-osk-no-uinput/releases) and copy to `~/.local/bin/`:

```bash
chmod +x gamepad-osk-no-uinput
mkdir -p ~/.local/bin
cp gamepad-osk-no-uinput ~/.local/bin/
```

**Option 2: Build in a distrobox.**

```bash
distrobox create --name build --image fedora:41
distrobox enter build
sudo dnf install golang SDL3-devel SDL3_ttf-devel wayland-devel wayland-protocols-devel
git clone https://github.com/0x90shell/gamepad-osk-no-uinput.git
cd gamepad-osk-no-uinput
go build -o gamepad-osk-no-uinput .
cp gamepad-osk-no-uinput ~/.local/bin/
exit
```

Make sure `~/.local/bin` is in your `$PATH`.

### Promptfont

[Promptfont](https://codeberg.org/shinmera/promptfont) displays controller button glyphs on mapped keys. Without it, those keys show text labels instead.

**Arch:**

```bash
yay -S ttf-promptfont
```

**All other distros:**

```bash
wget https://codeberg.org/shinmera/promptfont/releases/download/v1.14/promptfont.zip
unzip promptfont.zip promptfont.ttf
mkdir -p ~/.local/share/fonts
mv promptfont.ttf ~/.local/share/fonts/
fc-cache
rm promptfont.zip
```

## Configuration

Config is loaded from (first found):
1. `--config` flag
2. `~/.config/gamepad-osk-no-uinput/config`
3. `/etc/gamepad-osk-no-uinput/config`
4. `config` next to binary
5. `config` in working directory

A default config is auto-copied to `~/.config/gamepad-osk-no-uinput/` on first run.

See `config.example` for all options including button remapping, mouse stick, theme, scale, opacity, and deadzone.

## Themes

60 built-in themes. Cycle live with the Cfg key on the keyboard.

`ayu_dark` `candy` `catppuccin` `catppuccin_frappe` `cga` `chalk` `cobalt` `copper` `coral` `cyberpunk` `dark` `dracula` `ember` `everforest` `fjord` `forest` `gameboy` `gold` `gotham` `gruvbox` `high_contrast` `horizon` `ice` `kanagawa` `lavender` `material` `matrix` `mellow` `midnight` `monokai` `moss` `navy` `neon` `nightfox` `nord` `ocean` `olive` `onedark` `oxocarbon` `palenight` `paper` `plum` `retro` `rose_pine` `sakura` `sand` `slate` `solarized` `solarized_light` `steam_green` `sunset` `synthwave` `teal` `terminal` `tokyo_night` `tokyo_storm` `vapor` `virtualboy` `wine` `zx_spectrum`

| | | |
|:---:|:---:|:---:|
| ![ayu_dark](assets/ayu_dark.png)<br><sub>ayu_dark</sub> | ![candy](assets/candy.png)<br><sub>candy</sub> | ![catppuccin](assets/catppuccin.png)<br><sub>catppuccin</sub> |
| ![catppuccin_frappe](assets/catppuccin_frappe.png)<br><sub>catppuccin_frappe</sub> | ![cga](assets/cga.png)<br><sub>cga</sub> | ![chalk](assets/chalk.png)<br><sub>chalk</sub> |
| ![cobalt](assets/cobalt.png)<br><sub>cobalt</sub> | ![copper](assets/copper.png)<br><sub>copper</sub> | ![coral](assets/coral.png)<br><sub>coral</sub> |
| ![cyberpunk](assets/cyberpunk.png)<br><sub>cyberpunk</sub> | ![dark](assets/dark.png)<br><sub>dark</sub> | ![dracula](assets/dracula.png)<br><sub>dracula</sub> |
| ![ember](assets/ember.png)<br><sub>ember</sub> | ![everforest](assets/everforest.png)<br><sub>everforest</sub> | ![fjord](assets/fjord.png)<br><sub>fjord</sub> |
| ![forest](assets/forest.png)<br><sub>forest</sub> | ![gameboy](assets/gameboy.png)<br><sub>gameboy</sub> | ![gold](assets/gold.png)<br><sub>gold</sub> |
| ![gotham](assets/gotham.png)<br><sub>gotham</sub> | ![gruvbox](assets/gruvbox.png)<br><sub>gruvbox</sub> | ![high_contrast](assets/high_contrast.png)<br><sub>high_contrast</sub> |
| ![horizon](assets/horizon.png)<br><sub>horizon</sub> | ![ice](assets/ice.png)<br><sub>ice</sub> | ![kanagawa](assets/kanagawa.png)<br><sub>kanagawa</sub> |
| ![lavender](assets/lavender.png)<br><sub>lavender</sub> | ![material](assets/material.png)<br><sub>material</sub> | ![matrix](assets/matrix.png)<br><sub>matrix</sub> |
| ![mellow](assets/mellow.png)<br><sub>mellow</sub> | ![midnight](assets/midnight.png)<br><sub>midnight</sub> | ![monokai](assets/monokai.png)<br><sub>monokai</sub> |
| ![moss](assets/moss.png)<br><sub>moss</sub> | ![navy](assets/navy.png)<br><sub>navy</sub> | ![neon](assets/neon.png)<br><sub>neon</sub> |
| ![nightfox](assets/nightfox.png)<br><sub>nightfox</sub> | ![nord](assets/nord.png)<br><sub>nord</sub> | ![ocean](assets/ocean.png)<br><sub>ocean</sub> |
| ![olive](assets/olive.png)<br><sub>olive</sub> | ![onedark](assets/onedark.png)<br><sub>onedark</sub> | ![oxocarbon](assets/oxocarbon.png)<br><sub>oxocarbon</sub> |
| ![palenight](assets/palenight.png)<br><sub>palenight</sub> | ![paper](assets/paper.png)<br><sub>paper</sub> | ![plum](assets/plum.png)<br><sub>plum</sub> |
| ![retro](assets/retro.png)<br><sub>retro</sub> | ![rose_pine](assets/rose_pine.png)<br><sub>rose_pine</sub> | ![sakura](assets/sakura.png)<br><sub>sakura</sub> |
| ![sand](assets/sand.png)<br><sub>sand</sub> | ![slate](assets/slate.png)<br><sub>slate</sub> | ![solarized](assets/solarized.png)<br><sub>solarized</sub> |
| ![solarized_light](assets/solarized_light.png)<br><sub>solarized_light</sub> | ![steam_green](assets/steam_green.png)<br><sub>steam_green</sub> | ![sunset](assets/sunset.png)<br><sub>sunset</sub> |
| ![synthwave](assets/synthwave.png)<br><sub>synthwave</sub> | ![teal](assets/teal.png)<br><sub>teal</sub> | ![terminal](assets/terminal.png)<br><sub>terminal</sub> |
| ![tokyo_night](assets/tokyo_night.png)<br><sub>tokyo_night</sub> | ![tokyo_storm](assets/tokyo_storm.png)<br><sub>tokyo_storm</sub> | ![vapor](assets/vapor.png)<br><sub>vapor</sub> |
| ![virtualboy](assets/virtualboy.png)<br><sub>virtualboy</sub> | ![wine](assets/wine.png)<br><sub>wine</sub> | ![zx_spectrum](assets/zx_spectrum.png)<br><sub>zx_spectrum</sub> |

## Show & Hide

```ini
[gamepad]
combo_period_ms = 200        # timing window (ms)
```

Available buttons: `a`, `b`, `x`, `y`, `lb`, `rb`, `lt`, `rt`, `l3`, `r3`, `start`, `select`, `guide`, `dpad_up`, `dpad_down`, `dpad_left`, `dpad_right`. Requires 2-4 buttons.

### Coexistence with other gamepad readers

When the OSK becomes visible, gamepad-osk-no-uinput waits until no buttons are physically held before calling `EVIOCGRAB`. This means: any other process reading the same evdev node (evsieve, antimicroX, Steam Input, custom scripts) will observe the release events for the buttons that triggered the OSK before gamepad-osk-no-uinput takes over the device. The deferral is bounded at 500 ms — if the user holds the trigger combo longer than that, gamepad-osk-no-uinput grabs anyway. This mirrors evsieve's own `grab=auto` startup discipline.

### Device Grab

When `grab = true` (default), the gamepad is exclusively grabbed while the keyboard is visible. This prevents controller input from bleeding into the game while you're typing. The grab is released when the keyboard hides.

Note: `EVIOCGRAB` is per-evdev-node. If a consumer of gamepad input (e.g., an emulator, ES-DE, Moonlight) reads from `/dev/input/jsX` (legacy joydev) or from a sibling `event*` node belonging to the same physical controller, it will not be affected by the grab on the primary node. If you observe input bleeding through despite `grab = true`, check what fds the consumer has open (`ls -la /proc/<pid>/fd/ | grep input`) and either disable joydev fallback in that consumer (e.g., `SDL_JOYSTICK_LINUX_CLASSIC=0`) or tighten the device path to a single specific event node.

## Wayland

Uses wlr-layer-shell for native overlay support - no compositor rules needed. The keyboard renders as a non-focusable overlay that stays above all windows. Position toggle (Start) works natively.

**Supported compositors:**
- **wlroots-based** (Sway, wayfire, river, labwc, dwl) - native
- **Hyprland** - native
- **KDE Plasma 6** - native (via Layer Shell Qt)
- **COSMIC (Pop!_OS)** - native (via Smithay)
- **GNOME (Mutter)** - no layer-shell support; falls back to standard window (same as X11 without hints)

**Wayland limitations:**
- Pointer warp on hide is not available (Wayland does not allow clients to move the cursor). If you move the mouse to another monitor with the right stick, you need to move it back before hiding.
- Window opacity (`opacity` config) is not supported. SDL_SetWindowOpacity is a no-op on Wayland. See TODO.md for planned client-side fallback.

## License

MIT
