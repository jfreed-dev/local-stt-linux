package tray

import (
	"log"

	"github.com/getlantern/systray"
	"github.com/jfreed-dev/local-stt-linux/internal/mode"
)

// Callbacks for tray menu actions.
type Callbacks struct {
	OnModeChange func(mode.Mode)
	OnToggle     func()
	OnQuit       func()
}

// State represents the current app state for the tray.
type State struct {
	Mode      mode.Mode
	Recording bool
	Connected bool
}

var (
	callbacks Callbacks
	mPTT      *systray.MenuItem
	mVOX      *systray.MenuItem
	mAlways   *systray.MenuItem
	mToggle   *systray.MenuItem
	mStatus   *systray.MenuItem
)

// Run starts the system tray. This blocks and must be called from the main goroutine.
func Run(cb Callbacks) {
	callbacks = cb
	systray.Run(onReady, onExit)
}

// Quit exits the system tray.
func Quit() {
	systray.Quit()
}

// UpdateState updates the tray display based on current state.
func UpdateState(s State) {
	if mStatus == nil {
		return
	}

	status := "Idle"
	if s.Recording {
		status = "Recording"
	}
	if !s.Connected {
		status = "Disconnected"
	}
	mStatus.SetTitle(status)

	title := "STT"
	switch {
	case s.Recording:
		title = "STT [REC]"
		systray.SetIcon(iconRecording)
	case !s.Connected:
		title = "STT [!]"
		systray.SetIcon(iconError)
	default:
		systray.SetIcon(iconIdle)
	}
	systray.SetTitle(title)
	systray.SetTooltip("local-stt: " + status + " (" + string(s.Mode) + ")")

	// Update mode radio
	mPTT.Uncheck()
	mVOX.Uncheck()
	mAlways.Uncheck()
	switch s.Mode {
	case mode.PTT:
		mPTT.Check()
	case mode.VOX:
		mVOX.Check()
	case mode.Always:
		mAlways.Check()
	}
}

func onReady() {
	systray.SetIcon(iconIdle)
	systray.SetTitle("STT")
	systray.SetTooltip("local-stt: Speech-to-Text Dictation")

	mStatus = systray.AddMenuItem("Idle", "Current status")
	mStatus.Disable()
	systray.AddSeparator()

	mPTT = systray.AddMenuItemCheckbox("Push-to-Talk", "Hold hotkey to dictate", true)
	mVOX = systray.AddMenuItemCheckbox("VOX (Auto)", "Voice-activated dictation", false)
	mAlways = systray.AddMenuItemCheckbox("Always On", "Continuous dictation", false)
	systray.AddSeparator()

	mToggle = systray.AddMenuItem("Toggle On/Off", "Enable or disable dictation")
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Exit local-stt")

	go func() {
		for {
			select {
			case <-mPTT.ClickedCh:
				if callbacks.OnModeChange != nil {
					callbacks.OnModeChange(mode.PTT)
				}
			case <-mVOX.ClickedCh:
				if callbacks.OnModeChange != nil {
					callbacks.OnModeChange(mode.VOX)
				}
			case <-mAlways.ClickedCh:
				if callbacks.OnModeChange != nil {
					callbacks.OnModeChange(mode.Always)
				}
			case <-mToggle.ClickedCh:
				if callbacks.OnToggle != nil {
					callbacks.OnToggle()
				}
			case <-mQuit.ClickedCh:
				if callbacks.OnQuit != nil {
					callbacks.OnQuit()
				}
				systray.Quit()
			}
		}
	}()

	log.Printf("tray: system tray ready")
}

func onExit() {
	log.Printf("tray: exiting")
}
