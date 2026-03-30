# local-stt-linux

Native Linux speech-to-text keyboard input. Captures microphone audio, streams it to an [Aria](https://github.com/jfreed-dev/aria) STT server on your network, and injects the transcribed text as keyboard input into any focused application.

## Features

- **Push-to-talk** -- hold a hotkey combo to dictate (default: `Ctrl+\`)
- **VOX mode** -- automatic voice activity detection (server-side VAD)
- **Always-on mode** -- continuous dictation
- **Modifier+key combos** -- hotkeys support modifier combinations (e.g. `KEY_LEFTCTRL+KEY_BACKSLASH`)
- **System tray** -- status display, mode switching, toggle on/off
- **Wayland + X11** -- keyboard injection via ydotool
- **NoiseTorch compatible** -- select the virtual mic as audio source
- **keyd compatible** -- works alongside key remapping daemons

## Requirements

- Linux with PipeWire or PulseAudio
- [ydotool](https://github.com/ReimuNotMoe/ydotool) installed and `ydotoold` running
- User in `input` group (for global hotkeys and ydotool)
- Aria STT server accessible on the network

## Quick Start

```bash
# Build
make build

# Copy example config and edit
mkdir -p ~/.config/local-stt-linux
cp config.example.toml ~/.config/local-stt-linux/config.toml
# Edit config.toml with your server URL and preferences

# Run
./bin/local-stt

# Run with debug logging
./bin/local-stt --verbose

# Run without system tray
./bin/local-stt --no-tray
```

## Setup

```bash
# Add yourself to the input group (for hotkeys and ydotool)
sudo ./scripts/setup-uinput.sh

# Log out and back in for group changes to take effect

# Verify ydotoold is running
pgrep ydotoold || systemctl --user start ydotoold
```

## Configuration

Config file: `~/.config/local-stt-linux/config.toml`

See [config.example.toml](config.example.toml) for all options.

Key settings:
- `server.url` -- WebSocket URL for your Aria STT server
- `audio.source` -- PulseAudio source name (leave empty for default, or set to NoiseTorch virtual mic name from `pactl list sources short`)
- `mode.default` -- `ptt`, `vox`, or `always`
- `hotkey.ptt_key` -- evdev key combo for push-to-talk (default: `KEY_LEFTCTRL+KEY_BACKSLASH`)
- `hotkey.toggle_key` -- evdev key to toggle dictation on/off (default: `KEY_PAUSE`)

### Hotkey format

Single keys or modifier+key combos using evdev key names:

```toml
# Single key
ptt_key = "KEY_F12"

# Modifier + key combo
ptt_key = "KEY_LEFTCTRL+KEY_BACKSLASH"
```

Common key names: `KEY_LEFTCTRL`, `KEY_RIGHTCTRL`, `KEY_LEFTALT`, `KEY_LEFTMETA`, `KEY_F1`-`KEY_F12`, `KEY_SCROLLLOCK`, `KEY_PAUSE`, `KEY_INSERT`, `KEY_MEDIA`.

Use `sudo evtest` to discover the evdev key name for any key on your keyboard.

### Modes

| Mode | Behavior |
|------|----------|
| `ptt` | Hold the PTT hotkey to record; release to stop and transcribe |
| `vox` | Audio streams continuously; the server's VAD detects speech boundaries |
| `always` | Continuous streaming and transcription without pause |

Switch modes via the system tray menu or the mode cycle hotkey.

## Architecture

```
Mic (PulseAudio) -> PCM chunks -> Mode Gate -> WebSocket -> Aria STT Server
                                                                 |
Keyboard (ydotool) <- final text <- transcript_final <-----------'
```

The app speaks the Aria firmware WebSocket protocol, connecting as a virtual device with session ID `linux-stt-{hostname}`. Audio is streamed as raw 16kHz 16-bit signed little-endian mono PCM. The server runs dual VAD (WebRTC + Silero), STT (MLX Whisper), and returns partial/final transcriptions.

### Components

| Package | Responsibility |
|---------|---------------|
| `internal/audio` | PulseAudio mic capture at 16kHz mono PCM |
| `internal/stt` | WebSocket client implementing Aria firmware protocol |
| `internal/inject` | Keyboard injection via ydotool, wtype, or xdotool |
| `internal/hotkey` | Global hotkey listener via evdev with modifier combo support |
| `internal/mode` | PTT / VOX / always-on mode manager |
| `internal/tray` | System tray with ayatana-appindicator |
| `internal/config` | TOML config loader with defaults |

### Hotkey device filtering

The hotkey listener uses `ioctl EVIOCGBIT` to filter `/dev/input/event*` devices to actual keyboards (devices that support `KEY_A`). This avoids monitoring non-keyboard devices like lid switches and power buttons. Works alongside key remapping daemons like [keyd](https://github.com/rvaiya/keyd) by monitoring both physical and virtual keyboard devices.

## Versioning

This project uses [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release history.

Releases are created by tagging: `git tag v0.2.0 && git push --tags`. GitHub Actions builds and publishes binaries automatically.

## License

MIT
