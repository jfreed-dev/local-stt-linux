package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jfreed-dev/local-stt-linux/internal/audio"
	"github.com/jfreed-dev/local-stt-linux/internal/config"
	"github.com/jfreed-dev/local-stt-linux/internal/hotkey"
	"github.com/jfreed-dev/local-stt-linux/internal/inject"
	"github.com/jfreed-dev/local-stt-linux/internal/mode"
	"github.com/jfreed-dev/local-stt-linux/internal/postproc"
	"github.com/jfreed-dev/local-stt-linux/internal/stt"
	"github.com/jfreed-dev/local-stt-linux/internal/tray"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to config.toml")
	noTray := flag.Bool("no-tray", false, "disable system tray icon")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("local-stt", version)
		os.Exit(0)
	}

	if !*verbose {
		log.SetFlags(log.Ltime)
	} else {
		log.SetFlags(log.Ltime | log.Lshortfile)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	// Channels
	pcmCh := make(chan []byte, 64)
	hotkeyCh := make(chan hotkey.Event, 16)
	streamCh := make(chan stt.StreamEvent, 64)
	partialCh := make(chan string, 8)
	finalCh := make(chan string, 8)

	// Audio capture
	capturer := audio.NewCapturer(cfg.Audio.Source, cfg.Audio.SampleRate, cfg.Audio.ChunkMs, pcmCh)

	// Hotkey listener
	hotkeyListener := hotkey.NewListener(cfg.Hotkey.PTTKey, cfg.Hotkey.ToggleKey, hotkeyCh)

	// STT client
	sttClient := stt.NewClient(cfg.Server.URL, cfg.Server.InsecureTLS, streamCh, partialCh, finalCh)

	// LLM post-processor (corrects homophones, grammar, punctuation)
	proc := postproc.NewProcessor(cfg.PostProc.Endpoint, cfg.PostProc.Model, cfg.PostProc.Enabled)
	correctedCh := make(chan string, 8)

	// Keyboard injector reads from correctedCh (post-processed text)
	injector := inject.NewInjector(cfg.Inject.Method, cfg.Inject.TrailingSpace, cfg.Inject.AutoCap, cfg.Inject.InjectDelayMs, correctedCh)

	// Mode manager
	var onStatus mode.StatusFunc
	if !*noTray && cfg.Tray.Enabled {
		onStatus = func(m mode.Mode, recording bool) {
			tray.UpdateState(tray.State{Mode: m, Recording: recording, Connected: true})
		}
	}
	modeManager := mode.NewManager(cfg.Mode.Default, pcmCh, hotkeyCh, streamCh, onStatus)

	// Post-processing pipeline: finalCh -> LLM correction -> correctedCh
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case text, ok := <-finalCh:
				if !ok {
					return
				}
				corrected := proc.Process(ctx, text)
				select {
				case correctedCh <- corrected:
				default:
				}
			}
		}
	}()

	// Partial transcript logging
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case text, ok := <-partialCh:
				if !ok {
					return
				}
				if *verbose {
					log.Printf("partial: %s", text)
				}
			}
		}
	}()

	// Start background goroutines
	errCh := make(chan error, 5)

	go func() { errCh <- capturer.Run(ctx) }()
	go func() { errCh <- hotkeyListener.Run(ctx) }()
	go func() { errCh <- sttClient.Run(ctx) }()
	go func() { errCh <- injector.Run(ctx) }()
	go func() { errCh <- modeManager.Run(ctx) }()

	log.Printf("local-stt %s started (mode=%s, server=%s, postproc=%v)", version, cfg.Mode.Default, cfg.Server.URL, cfg.PostProc.Enabled)

	// System tray (blocks on main thread) or wait for shutdown
	if !*noTray && cfg.Tray.Enabled {
		tray.Run(tray.Callbacks{
			OnModeChange: func(m mode.Mode) {
				modeManager.SetMode(m)
			},
			OnToggle: func() {
				modeManager.Toggle()
			},
			OnQuit: func() {
				cancel()
			},
		})
	} else {
		// No tray -- wait for error or signal
		select {
		case err := <-errCh:
			if err != nil && ctx.Err() == nil {
				log.Fatalf("fatal: %v", err)
			}
		case <-ctx.Done():
		}
	}
}
