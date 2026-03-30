package inject

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

// Injector receives transcribed text and types it via keyboard simulation.
type Injector struct {
	method        string
	trailingSpace bool
	autoCap       bool
	delayMs       int
	finalCh       <-chan string
}

func NewInjector(method string, trailingSpace, autoCap bool, delayMs int, finalCh <-chan string) *Injector {
	return &Injector{
		method:        method,
		trailingSpace: trailingSpace,
		autoCap:       autoCap,
		delayMs:       delayMs,
		finalCh:       finalCh,
	}
}

// Run processes transcription results until ctx is cancelled.
func (inj *Injector) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case text, ok := <-inj.finalCh:
			if !ok {
				return nil
			}
			text = inj.processText(text)
			if text == "" {
				continue
			}

			if inj.delayMs > 0 {
				time.Sleep(time.Duration(inj.delayMs) * time.Millisecond)
			}

			if err := inj.typeText(ctx, text); err != nil {
				log.Printf("inject: %v", err)
			}
		}
	}
}

func (inj *Injector) processText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if inj.autoCap {
		runes := []rune(text)
		runes[0] = unicode.ToUpper(runes[0])
		text = string(runes)
	}
	if inj.trailingSpace {
		text += " "
	}
	return text
}

func (inj *Injector) typeText(ctx context.Context, text string) error {
	switch inj.method {
	case "ydotool":
		return runCmd(ctx, "ydotool", "type", "--", text)
	case "wtype":
		return runCmd(ctx, "wtype", text)
	case "xdotool":
		return runCmd(ctx, "xdotool", "type", "--", text)
	default:
		return fmt.Errorf("unknown inject method: %s", inj.method)
	}
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, string(out))
	}
	return nil
}
