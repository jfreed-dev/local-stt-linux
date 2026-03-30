# local-stt-linux

Native Linux speech-to-text keyboard input. Captures microphone audio, streams it to an [Aria](https://github.com/jfreed-dev/aria) STT server on your network, and injects the transcribed text as keyboard input into any focused application.

## Features

- **Push-to-talk** -- hold a hotkey to dictate
- **VOX mode** -- automatic voice activity detection (server-side)
- **Always-on mode** -- continuous dictation
- **System tray** -- status icons, mode switching, audio source selection
- **Wayland + X11** -- keyboard injection via ydotool
- **NoiseTorch compatible** -- select the virtual mic as audio source

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
```

## Setup

```bash
# Add yourself to the input group (for hotkeys and ydotool)
sudo ./scripts/setup-uinput.sh

# Log out and back in for group changes to take effect
```

## Configuration

See [config.example.toml](config.example.toml) for all options.

Key settings:
- `server.url` -- WebSocket URL for your Aria STT server
- `audio.source` -- PulseAudio source name (leave empty for default, or set to NoiseTorch virtual mic)
- `mode.default` -- `ptt`, `vox`, or `always`
- `hotkey.ptt_key` -- evdev key name for push-to-talk (default: `KEY_SCROLLLOCK`)

## Architecture

```
Mic (PulseAudio) -> PCM chunks -> Mode Gate -> WebSocket -> Aria STT Server
                                                                 |
Keyboard (ydotool) <- final text <- transcript_final <-----------'
```

The app speaks the Aria firmware WebSocket protocol, connecting as a virtual device. Audio is streamed as raw 16kHz 16-bit mono PCM. The server runs VAD, STT (MLX Whisper), and returns partial/final transcriptions.

## Versioning

This project uses [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release history.

Releases are created by tagging: `git tag v0.1.0 && git push --tags`. GitHub Actions builds and publishes binaries automatically.

## License

MIT
