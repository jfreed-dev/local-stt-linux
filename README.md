# local-stt-linux

Native Linux speech-to-text keyboard input. Captures microphone audio, streams it to an [Aria](https://github.com/jfreed-dev/aria) STT server on your network, and injects the transcribed text as keyboard input into any focused application.

## Features

- **Push-to-talk** -- hold a hotkey combo to dictate (default: `Ctrl+\`)
- **VOX mode** -- automatic voice activity detection (server-side VAD)
- **Always-on mode** -- continuous dictation
- **LLM post-processing** -- fixes homophones, grammar, and punctuation via a local LLM before typing
- **Audio feedback** -- chirp sounds on PTT start/stop (freedesktop system sounds)
- **Modifier+key combos** -- hotkeys support modifier combinations (e.g. `KEY_LEFTCTRL+KEY_BACKSLASH`)
- **Custom tray icons** -- microphone icons reflect state (idle/recording/error)
- **System tray** -- status display, mode switching, toggle on/off
- **Wayland + X11** -- keyboard injection via ydotool
- **NoiseTorch compatible** -- select the virtual mic as audio source
- **keyd compatible** -- works alongside key remapping daemons

## Requirements

- Linux with PipeWire or PulseAudio
- [ydotool](https://github.com/ReimuNotMoe/ydotool) installed and `ydotoold` running
- User in `input` group (for global hotkeys and ydotool)
- Aria STT server accessible on the network
- (Optional) Local LLM endpoint for post-processing (OpenAI-compatible API)

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

### CLI Flags

| Flag | Description |
|------|-------------|
| `--config PATH` | Path to config.toml (default: `~/.config/local-stt-linux/config.toml`) |
| `--verbose` | Enable debug logging (partial transcripts, hotkey events, postproc diffs) |
| `--no-tray` | Run without system tray icon |
| `--version` | Print version and exit |

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

### Server

| Setting | Description | Default |
|---------|-------------|---------|
| `server.url` | Aria STT server WebSocket URL | `ws://localhost:5182/ws/firmware` |
| `server.insecure_tls` | Skip TLS cert verification | `true` |

### Audio

| Setting | Description | Default |
|---------|-------------|---------|
| `audio.source` | PulseAudio source name (empty = default mic) | `""` |
| `audio.sample_rate` | Sample rate in Hz (must be 16000) | `16000` |
| `audio.chunk_ms` | Audio chunk duration in ms | `100` |

Find available sources with `pactl list sources short`. Set to the NoiseTorch virtual mic name for noise cancellation.

### Hotkeys

| Setting | Description | Default |
|---------|-------------|---------|
| `hotkey.ptt_key` | Push-to-talk key combo | `KEY_LEFTCTRL+KEY_BACKSLASH` |
| `hotkey.toggle_key` | Toggle dictation on/off | `KEY_PAUSE` |

Single keys or modifier+key combos using evdev key names:

```toml
# Single key
ptt_key = "KEY_F12"

# Modifier + key combo (Ctrl+\)
ptt_key = "KEY_LEFTCTRL+KEY_BACKSLASH"
```

Common key names: `KEY_LEFTCTRL`, `KEY_RIGHTCTRL`, `KEY_LEFTALT`, `KEY_LEFTMETA`, `KEY_F1`-`KEY_F12`, `KEY_SCROLLLOCK`, `KEY_PAUSE`, `KEY_INSERT`, `KEY_MEDIA`.

Use `sudo evtest` to discover the evdev key name for any key on your keyboard. Note: if you use [keyd](https://github.com/rvaiya/keyd) or another key remapper, check the virtual keyboard device output, not the physical one.

### Modes

| Mode | Behavior |
|------|----------|
| `ptt` | Hold the PTT hotkey to record; release to stop and transcribe |
| `vox` | Audio streams continuously; the server's VAD detects speech boundaries |
| `always` | Continuous streaming and transcription without pause |

Switch modes via the system tray menu or the mode cycle hotkey.

### Keyboard Injection

| Setting | Description | Default |
|---------|-------------|---------|
| `inject.method` | Injection tool: `ydotool`, `wtype`, `xdotool` | `ydotool` |
| `inject.trailing_space` | Add space after each injection | `true` |
| `inject.auto_capitalize` | Capitalize first word | `true` |
| `inject.inject_delay_ms` | Per-keystroke delay for ydotool (ms) | `25` |

### LLM Post-Processing

Raw STT output can contain homophone errors ("right" vs "write") since Whisper transcribes phonetically without context. The post-processor sends transcriptions through a local LLM for correction before typing.

| Setting | Description | Default |
|---------|-------------|---------|
| `postproc.enabled` | Enable LLM correction | `false` |
| `postproc.endpoint` | OpenAI-compatible API base URL | `http://localhost:8003` |
| `postproc.model` | Model name served at the endpoint | `mlx-community/Mistral-Nemo-Instruct-2407-4bit` |

The post-processor adds ~2-3 seconds of latency. It gracefully falls back to raw text if the LLM is unavailable. Enable with `--verbose` to see correction diffs in the log.

### System Tray

| Setting | Description | Default |
|---------|-------------|---------|
| `tray.enabled` | Show system tray icon | `true` |

Tray icon states:
- **Gray microphone** -- idle, waiting for PTT or input
- **Green microphone + red dot** -- actively recording
- **Red microphone** -- disconnected from server

## Architecture

```
Mic (PulseAudio) -> PCM chunks -> Mode Gate -> WebSocket -> Aria STT Server
                                                                 |
                                                        transcript_final
                                                                 |
                                                          LLM Post-Proc
                                                          (fix grammar)
                                                                 |
Keyboard (ydotool) <------ corrected text <----------------------'
```

The app speaks the Aria firmware WebSocket protocol, connecting as a virtual device with session ID `linux-stt-{hostname}`. Audio is streamed as raw 16kHz 16-bit signed little-endian mono PCM. The server runs dual VAD (WebRTC + Silero), STT (MLX Whisper), and returns partial/final transcriptions.

### Components

| Package | Responsibility |
|---------|---------------|
| `internal/audio` | PulseAudio mic capture at 16kHz mono PCM |
| `internal/stt` | WebSocket client implementing Aria firmware protocol |
| `internal/postproc` | LLM post-processing for grammar/homophone correction |
| `internal/inject` | Keyboard injection via ydotool, wtype, or xdotool |
| `internal/hotkey` | Global hotkey listener via evdev with modifier combo support |
| `internal/mode` | PTT / VOX / always-on mode manager |
| `internal/sound` | PTT audio feedback via freedesktop system sounds |
| `internal/tray` | System tray with ayatana-appindicator and generated icons |
| `internal/config` | TOML config loader with defaults |

### Concurrency

```
audio.Capture  --> pcmCh ----> mode.Manager --> streamCh --> stt.Client
hotkey.Listen  --> hotkeyCh ->      |                            |
                                    |                       finalCh
                                    |                            |
                              sound.Play*()               postproc.Process
                                                                 |
                                                           correctedCh
                                                                 |
                                                          inject.Injector
                                                                 |
                                                         ydotool type --
```

All components communicate via buffered Go channels. The mode manager is the central coordinator that gates audio based on hotkey state and current mode.

### Hotkey device filtering

The hotkey listener uses `ioctl EVIOCGBIT` to filter `/dev/input/event*` devices to actual keyboards (devices that support `KEY_A`). This avoids monitoring non-keyboard devices like lid switches and power buttons. Works alongside key remapping daemons like [keyd](https://github.com/rvaiya/keyd) by monitoring both physical and virtual keyboard devices.

### Aria server authorization

The client connects with session ID `linux-stt-{hostname}`. The Aria server's `firmware_auth` in `config/security.json` must include `linux-stt-*` in `allowed_device_ids`, and `guardian/filters.py:is_device_authorized()` must support wildcard suffix matching.

## Versioning

This project uses [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release history.

Releases are created by tagging: `git tag v0.2.0 && git push --tags`. GitHub Actions builds and publishes binaries automatically.

## License

MIT
