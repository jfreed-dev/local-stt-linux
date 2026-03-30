# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Modifier+key combo hotkeys (e.g. `KEY_LEFTCTRL+KEY_BACKSLASH`)
- Smart keyboard device filtering via ioctl EVIOCGBIT (only monitors actual keyboards)
- epoll-based hotkey reads with proper context cancellation
- Verbose hotkey logging with device path for debugging
- Extended evdev key map: media keys, KEY_LEFTCTRL, KEY_BACKSLASH, KEY_LEFTALT, KEY_SPACE

### Changed
- Default PTT hotkey changed from KEY_SCROLLLOCK to KEY_LEFTCTRL+KEY_BACKSLASH (Ctrl+\)
- Hotkey listener now filters to ~3 keyboard devices instead of monitoring all 16+ input devices

### Fixed
- System tray mode switching and toggle now actually change the mode (were no-ops before)
- .gitignore no longer blocks cmd/local-stt/ directory

## [0.1.0] - 2026-03-30

### Added
- Project scaffolding with Go module, Makefile, CI workflow
- TOML configuration with sensible defaults (`~/.config/local-stt-linux/config.toml`)
- PulseAudio audio capture at 16kHz mono PCM (PipeWire compatible)
- WebSocket client implementing Aria firmware protocol (hello, stream start/chunk/end)
- Auto-reconnect with exponential backoff on disconnect
- Keyboard injection via ydotool (Wayland/X11), wtype, or xdotool
- Push-to-talk mode with configurable evdev hotkey
- VOX mode (continuous streaming with server-side VAD)
- Always-on dictation mode
- Global hotkey listener via /dev/input/event* (evdev)
- Mode cycling and global toggle hotkeys
- System tray with status display, mode switching, and quit menu
- Auto-capitalize first word of each dictation
- Configurable trailing space after injection
- NoiseTorch virtual mic support (select by PulseAudio source name)
- Setup script for /dev/uinput permissions and ydotoold
- GitHub Actions CI for linux/amd64 and linux/arm64
- Initial project creation
