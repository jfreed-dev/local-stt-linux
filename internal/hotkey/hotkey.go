package hotkey

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
	evKey      = 0x01
	keyPress   = 1
	keyRelease = 0

	// ioctl to test if a key code is supported by the device
	eviocgbit0 = 0x80084520 // EVIOCGBIT(EV_SYN, 32)
	eviocgbit1 = 0x80604521 // EVIOCGBIT(EV_KEY, 96*8)
)

// hotCombo represents a key combination (modifier + key).
type hotCombo struct {
	modifier uint16 // 0 = no modifier required
	key      uint16
}

// Listener watches /dev/input/event* for configured hotkeys.
type Listener struct {
	pttCombo    hotCombo
	toggleCombo hotCombo
	eventCh     chan<- Event
}

func NewListener(pttKeyName, toggleKeyName string, eventCh chan<- Event) *Listener {
	return &Listener{
		pttCombo:    parseCombo(pttKeyName),
		toggleCombo: parseCombo(toggleKeyName),
		eventCh:     eventCh,
	}
}

// parseCombo parses "KEY_LEFTCTRL+KEY_BACKSLASH" or "KEY_F12" into a hotCombo.
func parseCombo(name string) hotCombo {
	parts := strings.Split(name, "+")
	if len(parts) == 2 {
		return hotCombo{
			modifier: keyNameToCode(parts[0]),
			key:      keyNameToCode(parts[1]),
		}
	}
	return hotCombo{key: keyNameToCode(name)}
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

	log.Printf("hotkey: monitoring %d keyboard device(s), ptt=%s toggle=%s",
		len(devices), comboStr(l.pttCombo), comboStr(l.toggleCombo))
	for _, d := range devices {
		log.Printf("hotkey:   %s", d)
	}

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
		log.Printf("hotkey: skipping %s: %v", path, err)
		return nil
	}
	defer f.Close()

	fd := int(f.Fd())
	evSize := int(unsafe.Sizeof(inputEvent{}))
	buf := make([]byte, evSize)

	// Use epoll to make reads cancellable via context
	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		return fmt.Errorf("epoll_create: %w", err)
	}
	defer syscall.Close(epfd)

	err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	})
	if err != nil {
		return fmt.Errorf("epoll_ctl: %w", err)
	}

	events := make([]syscall.EpollEvent, 1)

	// Track held keys for modifier combos
	heldKeys := make(map[uint16]bool)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Wait up to 500ms for data
		n, err := syscall.EpollWait(epfd, events, 500)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return fmt.Errorf("epoll_wait %s: %w", path, err)
		}
		if n == 0 {
			continue // timeout, check ctx
		}

		nread, err := f.Read(buf)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if nread < evSize {
			continue
		}

		var ev inputEvent
		ev.Type = binary.LittleEndian.Uint16(buf[16:18])
		ev.Code = binary.LittleEndian.Uint16(buf[18:20])
		ev.Value = int32(binary.LittleEndian.Uint32(buf[20:24]))

		if ev.Type != evKey {
			continue
		}

		// Track key state
		if ev.Value == keyPress {
			heldKeys[ev.Code] = true
		} else if ev.Value == keyRelease {
			delete(heldKeys, ev.Code)
		}

		// Check combos
		for _, combo := range []struct {
			name string
			c    hotCombo
		}{
			{"ptt", l.pttCombo},
			{"toggle", l.toggleCombo},
		} {
			if ev.Code != combo.c.key {
				continue
			}
			// If combo has a modifier, check it's held
			if combo.c.modifier != 0 && !heldKeys[combo.c.modifier] {
				continue
			}
			if ev.Value == keyPress || ev.Value == keyRelease {
				pressed := ev.Value == keyPress
				log.Printf("hotkey: %s %s (code=%d) from %s",
					combo.name, map[bool]string{true: "pressed", false: "released"}[pressed], ev.Code, path)
				select {
				case l.eventCh <- Event{Key: combo.name, Pressed: pressed}:
				default:
				}
			}
		}
	}
}

// findKeyboardDevices returns event devices that support EV_KEY with common keyboard keys.
func findKeyboardDevices() ([]string, error) {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, err
	}

	var keyboards []string
	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		if isKeyboard(f) {
			keyboards = append(keyboards, path)
		}
		f.Close()
	}
	return keyboards, nil
}

// isKeyboard checks if the device supports EV_KEY with KEY_A (code 30).
// This filters out non-keyboard devices like lid switches, power buttons, etc.
func isKeyboard(f *os.File) bool {
	// Check which event types are supported
	var evBits [4]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(),
		uintptr(eviocgbit0), uintptr(unsafe.Pointer(&evBits[0])))
	if errno != 0 {
		return false
	}
	// Check EV_KEY bit (bit 1)
	if evBits[0]&(1<<evKey) == 0 {
		return false
	}

	// Check if KEY_A (code 30) is supported -- filters out non-keyboard devices
	var keyBits [96]byte // 768 bits covers all standard keys
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(),
		uintptr(eviocgbit1), uintptr(unsafe.Pointer(&keyBits[0])))
	if errno != 0 {
		return false
	}
	// KEY_A = 30, byte 30/8=3, bit 30%8=6
	return keyBits[3]&(1<<6) != 0
}

func comboStr(c hotCombo) string {
	mod := ""
	key := fmt.Sprintf("0x%x", c.key)
	for name, code := range keyMap {
		if code == c.key {
			key = name
		}
		if c.modifier != 0 && code == c.modifier {
			mod = name + "+"
		}
	}
	return mod + key
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
	"KEY_BACKSLASH":  43,
	"KEY_LEFTALT":    56,
	"KEY_SPACE":      57,
	"KEY_CAPSLOCK":   58,
	"KEY_SYSRQ":     99,
	"KEY_LEFTCTRL":   29,
	"KEY_RIGHTCTRL":  97,
	"KEY_LEFTMETA":   125,
	"KEY_RIGHTMETA":  126,
	"KEY_COMPOSE":    127,
	"KEY_STOP":       128,
	"KEY_CALC":       140,
	"KEY_SLEEP":      142,
	"KEY_WAKEUP":     143,
	"KEY_MAIL":       155,
	"KEY_BOOKMARKS":  156,
	"KEY_COMPUTER":   157,
	"KEY_BACK":       158,
	"KEY_FORWARD":    159,
	"KEY_NEXTSONG":   163,
	"KEY_PLAYPAUSE":  164,
	"KEY_PREVIOUSSONG": 165,
	"KEY_STOPCD":     166,
	"KEY_HOMEPAGE":   172,
	"KEY_REFRESH":    173,
	"KEY_MEDIA":      226,
	"KEY_SEARCH":     217,
	"KEY_MUTE":       113,
	"KEY_VOLUMEDOWN": 114,
	"KEY_VOLUMEUP":   115,
}
