# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- LLM post-processing: sends raw STT text through a local LLM (OpenAI-compatible
  endpoint) to fix homophones, grammar, and punctuation before keyboard injection.
  Configurable endpoint/model, 15s timeout, graceful fallback to raw text on error.
  Disabled by default; enable in config with `[postproc] enabled = true`.
- PTT audio feedback: plays freedesktop system sounds (device-added.oga /
  device-removed.oga) on recording start/stop via pw-play or paplay.
- Custom system tray icons: programmatically generated 22x22 PNG microphone icons
  for idle (gray), recording (green + red dot), and error (red) states. No external
  image files needed -- icons are embedded in the binary.
- Modifier+key combo hotkeys (e.g. `KEY_LEFTCTRL+KEY_BACKSLASH`) with held-key
  tracking per device.
- Smart keyboard device filtering via ioctl EVIOCGBIT (only monitors devices that
  support KEY_A, reducing from ~16 to ~3 devices).
- epoll-based hotkey reads with proper context cancellation.
- Verbose hotkey logging with device path for debugging.
- Extended evdev key map: media keys, KEY_LEFTCTRL, KEY_BACKSLASH, KEY_LEFTALT,
  KEY_SPACE, and all function/media keys.

### Changed
- Default PTT hotkey changed from KEY_SCROLLLOCK to KEY_LEFTCTRL+KEY_BACKSLASH
  (Ctrl+\). Recommended: use a non-modifier key like KEY_F12 to avoid modifier
  bleed into ydotool injection.
- `inject_delay_ms` now controls ydotool's `--key-delay` (per-keystroke delay)
  instead of a single pre-injection sleep. Default changed from 50 to 25ms.
- Architecture diagram updated: data flow now includes LLM post-processing step
  between STT and keyboard injection.

### Added
- Timing logs for postproc round-trip and keystroke injection duration.

### Fixed
- System tray mode switching and toggle now actually change the mode (were no-ops).
- .gitignore no longer blocks cmd/local-stt/ directory.
- Modifier keys (Ctrl, Alt, Shift) no longer bleed into ydotool keystroke injection
  when using modifier+key PTT combos. Using a non-modifier PTT key (e.g. F12) is
  recommended to avoid this entirely.

## [0.1.0] - 2026-03-30

### Added
- Project scaffolding with Go module, Makefile, CI workflow.
- TOML configuration with sensible defaults (`~/.config/local-stt-linux/config.toml`).
- PulseAudio audio capture at 16kHz mono PCM (PipeWire compatible).
- WebSocket client implementing Aria firmware protocol (hello, stream start/chunk/end).
- Auto-reconnect with exponential backoff on disconnect.
- Keyboard injection via ydotool (Wayland/X11), wtype, or xdotool.
- Push-to-talk mode with configurable evdev hotkey.
- VOX mode (continuous streaming with server-side VAD).
- Always-on dictation mode.
- Global hotkey listener via /dev/input/event* (evdev).
- Mode cycling and global toggle hotkeys.
- System tray with status display, mode switching, and quit menu.
- Auto-capitalize first word of each dictation.
- Configurable trailing space after injection.
- NoiseTorch virtual mic support (select by PulseAudio source name).
- Setup script for /dev/uinput permissions and ydotoold.
- GitHub Actions CI for linux/amd64 and linux/arm64.
