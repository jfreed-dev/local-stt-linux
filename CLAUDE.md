# CLAUDE.md -- Project Instructions for Claude Code

## Token Efficiency (drona23/claude-token-efficient)

- Do not start responses with sycophantic openers ("Sure!", "Great question!", "Absolutely!")
- Do not end with hollow closings ("Hope this helps!", "Let me know if you need anything!")
- Do not restate the user's request -- execute immediately
- Use ASCII-only output: no em dashes, smart quotes, or Unicode decorations
- No "As an AI..." framing
- Only include safety-critical disclaimers -- skip unnecessary caveats
- Deliver exactly what was requested -- no unsolicited suggestions
- Provide minimal working solutions -- avoid over-engineered abstractions
- Say "I don't know" rather than guessing or hallucinating
- Treat user corrections as session ground truth
- Never re-read the same file twice in a session
- Do not modify code outside the explicit request scope

## Project-Specific Rules

- Language: Go. Follow standard Go idioms and conventions.
- Keep dependencies minimal -- only add what is justified.
- Test with `go test ./...` and build with `go build ./cmd/local-stt/`.
- The Aria STT server firmware WebSocket protocol is at `ws://<server-ip>:5182/ws/firmware`.
- Audio format: 16kHz, 16-bit signed little-endian mono PCM.
- Keyboard injection uses ydotool on Wayland (COSMIC desktop).
- NoiseTorch provides a virtual PulseAudio source for noise cancellation -- the app selects it by source name.
- Config lives at `~/.config/local-stt-linux/config.toml`.
