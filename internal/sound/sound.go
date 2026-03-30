package sound

import (
	"log"
	"os/exec"
)

const (
	startSound = "/usr/share/sounds/freedesktop/stereo/device-added.oga"
	stopSound  = "/usr/share/sounds/freedesktop/stereo/device-removed.oga"
)

// PlayStart plays the recording-started chirp.
func PlayStart() {
	play(startSound)
}

// PlayStop plays the recording-stopped chirp.
func PlayStop() {
	play(stopSound)
}

func play(path string) {
	// Fire and forget -- don't block the caller
	go func() {
		// Try pw-play first (PipeWire native), fall back to paplay
		for _, cmd := range []string{"pw-play", "paplay"} {
			if err := exec.Command(cmd, path).Run(); err == nil {
				return
			}
		}
		log.Printf("sound: could not play %s", path)
	}()
}
