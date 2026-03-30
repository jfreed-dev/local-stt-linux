package mode

import (
	"context"
	"log"
	"sync"

	"github.com/jfreed-dev/local-stt-linux/internal/hotkey"
	"github.com/jfreed-dev/local-stt-linux/internal/stt"
)

// Mode represents a dictation mode.
type Mode string

const (
	PTT    Mode = "ptt"    // Push-to-talk: stream while hotkey held
	VOX    Mode = "vox"    // Voice-activated: stream always, server VAD
	Always Mode = "always" // Always-on: continuous streaming
)

// StatusFunc is called when mode or recording state changes.
type StatusFunc func(mode Mode, recording bool)

// Manager coordinates audio gating based on the current mode and hotkey state.
type Manager struct {
	mu        sync.Mutex
	mode      Mode
	enabled   bool // global on/off toggle
	recording bool // currently streaming audio

	pcmCh    <-chan []byte
	hotkeyCh <-chan hotkey.Event
	streamCh chan<- stt.StreamEvent
	onStatus StatusFunc
}

func NewManager(defaultMode string, pcmCh <-chan []byte, hotkeyCh <-chan hotkey.Event, streamCh chan<- stt.StreamEvent, onStatus StatusFunc) *Manager {
	m := &Manager{
		mode:     Mode(defaultMode),
		enabled:  true,
		pcmCh:    pcmCh,
		hotkeyCh: hotkeyCh,
		streamCh: streamCh,
		onStatus: onStatus,
	}
	if m.mode != PTT && m.mode != VOX && m.mode != Always {
		m.mode = PTT
	}
	return m
}

// CycleMode switches to the next mode: PTT -> VOX -> Always -> PTT.
func (m *Manager) CycleMode() {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.mode {
	case PTT:
		m.mode = VOX
	case VOX:
		m.mode = Always
	case Always:
		m.mode = PTT
	}
	log.Printf("mode: switched to %s", m.mode)
	if m.onStatus != nil {
		m.onStatus(m.mode, m.recording)
	}
}

// Run processes hotkey events and audio, routing to the STT client.
func (m *Manager) Run(ctx context.Context) error {
	pttHeld := false

	// In VOX/Always modes, start streaming immediately when enabled
	m.mu.Lock()
	if m.enabled && (m.mode == VOX || m.mode == Always) {
		m.startRecording()
	}
	m.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			m.mu.Lock()
			if m.recording {
				m.stopRecording()
			}
			m.mu.Unlock()
			return ctx.Err()

		case evt, ok := <-m.hotkeyCh:
			if !ok {
				return nil
			}
			m.mu.Lock()
			switch evt.Key {
			case "ptt":
				pttHeld = evt.Pressed
				if m.mode == PTT && m.enabled {
					if evt.Pressed && !m.recording {
						m.startRecording()
					} else if !evt.Pressed && m.recording {
						m.stopRecording()
					}
				}
			case "toggle":
				if evt.Pressed {
					m.enabled = !m.enabled
					log.Printf("mode: dictation %s", map[bool]string{true: "enabled", false: "disabled"}[m.enabled])
					if !m.enabled && m.recording {
						m.stopRecording()
					} else if m.enabled && (m.mode == VOX || m.mode == Always) {
						m.startRecording()
					} else if m.enabled && m.mode == PTT && pttHeld {
						m.startRecording()
					}
					if m.onStatus != nil {
						m.onStatus(m.mode, m.recording)
					}
				}
			}
			m.mu.Unlock()

		case pcm, ok := <-m.pcmCh:
			if !ok {
				return nil
			}
			m.mu.Lock()
			if m.recording {
				select {
				case m.streamCh <- stt.StreamEvent{Type: "chunk", Data: pcm}:
				default:
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Manager) startRecording() {
	m.recording = true
	select {
	case m.streamCh <- stt.StreamEvent{Type: "start"}:
	default:
	}
	log.Printf("mode: recording started (%s)", m.mode)
	if m.onStatus != nil {
		m.onStatus(m.mode, true)
	}
}

func (m *Manager) stopRecording() {
	m.recording = false
	select {
	case m.streamCh <- stt.StreamEvent{Type: "end"}:
	default:
	}
	log.Printf("mode: recording stopped (%s)", m.mode)
	if m.onStatus != nil {
		m.onStatus(m.mode, false)
	}
}
