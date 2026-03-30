package hotkey

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

// Event represents a hotkey press or release.
type Event struct {
	Key     string
	Pressed bool
}

// inputEvent matches the Linux input_event struct.
type inputEvent struct {
	Time  [2]int64 // timeval: sec, usec
	Type  uint16
	Code  uint16
	Value int32
}

const (
	evKey     = 0x01
	keyPress  = 1
	keyRelease = 0
)

// Listener watches /dev/input/event* for configured hotkeys.
type Listener struct {
	pttKey    uint16
	toggleKey uint16
	eventCh   chan<- Event
}

func NewListener(pttKeyName, toggleKeyName string, eventCh chan<- Event) *Listener {
	return &Listener{
		pttKey:    keyNameToCode(pttKeyName),
		toggleKey: keyNameToCode(toggleKeyName),
		eventCh:   eventCh,
	}
}

// Run scans for keyboard devices and monitors them for hotkey events.
func (l *Listener) Run(ctx context.Context) error {
	devices, err := findKeyboardDevices()
	if err != nil {
		return fmt.Errorf("find keyboards: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no keyboard devices found in /dev/input/")
	}

	log.Printf("hotkey: monitoring %d device(s), ptt=0x%x toggle=0x%x",
		len(devices), l.pttKey, l.toggleKey)

	errCh := make(chan error, len(devices))
	for _, dev := range devices {
		go func(path string) {
			errCh <- l.monitorDevice(ctx, path)
		}(dev)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (l *Listener) monitorDevice(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	evSize := int(unsafe.Sizeof(inputEvent{}))
	buf := make([]byte, evSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set read deadline to avoid blocking forever
		f.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := f.Read(buf)
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", path, err)
		}
		if n < evSize {
			continue
		}

		var ev inputEvent
		ev.Type = binary.LittleEndian.Uint16(buf[16:18])
		ev.Code = binary.LittleEndian.Uint16(buf[18:20])
		ev.Value = int32(binary.LittleEndian.Uint32(buf[20:24]))

		if ev.Type != evKey {
			continue
		}

		var keyName string
		var matched bool
		if ev.Code == l.pttKey {
			keyName = "ptt"
			matched = true
		} else if ev.Code == l.toggleKey {
			keyName = "toggle"
			matched = true
		}

		if matched && (ev.Value == keyPress || ev.Value == keyRelease) {
			select {
			case l.eventCh <- Event{Key: keyName, Pressed: ev.Value == keyPress}:
			default:
			}
		}
	}
}

func findKeyboardDevices() ([]string, error) {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, err
	}
	// Return all event devices; filtering by capability would require ioctl
	// which adds complexity. Reading non-keyboard devices is harmless.
	var accessible []string
	for _, m := range matches {
		if f, err := os.Open(m); err == nil {
			f.Close()
			accessible = append(accessible, m)
		}
	}
	return accessible, nil
}

// keyNameToCode converts an evdev key name like "KEY_SCROLLLOCK" to its code.
func keyNameToCode(name string) uint16 {
	name = strings.ToUpper(strings.TrimSpace(name))
	if code, ok := keyMap[name]; ok {
		return code
	}
	log.Printf("hotkey: unknown key name %q, using 0", name)
	return 0
}

// Common evdev key codes.
var keyMap = map[string]uint16{
	"KEY_ESC":        1,
	"KEY_1":          2,
	"KEY_2":          3,
	"KEY_3":          4,
	"KEY_4":          5,
	"KEY_5":          6,
	"KEY_6":          7,
	"KEY_7":          8,
	"KEY_8":          9,
	"KEY_9":          10,
	"KEY_0":          11,
	"KEY_F1":         59,
	"KEY_F2":         60,
	"KEY_F3":         61,
	"KEY_F4":         62,
	"KEY_F5":         63,
	"KEY_F6":         64,
	"KEY_F7":         65,
	"KEY_F8":         66,
	"KEY_F9":         67,
	"KEY_F10":        68,
	"KEY_F11":        87,
	"KEY_F12":        88,
	"KEY_NUMLOCK":    69,
	"KEY_SCROLLLOCK": 70,
	"KEY_PAUSE":      119,
	"KEY_INSERT":     110,
	"KEY_HOME":       102,
	"KEY_PAGEUP":     104,
	"KEY_DELETE":     111,
	"KEY_END":        107,
	"KEY_PAGEDOWN":   109,
	"KEY_CAPSLOCK":   58,
	"KEY_SYSRQ":     99,
	"KEY_RIGHTCTRL":  97,
	"KEY_LEFTMETA":   125,
	"KEY_RIGHTMETA":  126,
	"KEY_COMPOSE":    127,
}
