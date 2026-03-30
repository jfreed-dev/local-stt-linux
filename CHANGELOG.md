# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project scaffolding with Go module, Makefile, CI workflow
- TOML configuration with sensible defaults (`~/.config/local-stt-linux/config.toml`)
- PulseAudio audio capture at 16kHz mono PCM (PipeWire compatible)
- WebSocket client implementing Aria firmware protocol (hello, stream start/chunk/end)
- Auto-reconnect with exponential backoff on disconnect
- Keyboard injection via ydotool (Wayland/X11), wtype, or xdotool
- Push-to-talk mode with configurable evdev hotkey (default: ScrollLock)
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

## [0.1.0] - 2026-03-30

### Added
- Initial project creation
