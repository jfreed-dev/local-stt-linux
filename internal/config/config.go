package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server ServerConfig `toml:"server"`
	Audio  AudioConfig  `toml:"audio"`
	Mode   ModeConfig   `toml:"mode"`
	Hotkey HotkeyConfig `toml:"hotkey"`
	Inject InjectConfig `toml:"inject"`
	Tray   TrayConfig   `toml:"tray"`
}

type ServerConfig struct {
	URL         string `toml:"url"`
	InsecureTLS bool   `toml:"insecure_tls"`
}

type AudioConfig struct {
	Source     string `toml:"source"`
	SampleRate int    `toml:"sample_rate"`
	ChunkMs    int    `toml:"chunk_ms"`
}

type ModeConfig struct {
	Default string `toml:"default"`
}

type HotkeyConfig struct {
	PTTKey    string `toml:"ptt_key"`
	ToggleKey string `toml:"toggle_key"`
}

type InjectConfig struct {
	Method        string `toml:"method"`
	TrailingSpace bool   `toml:"trailing_space"`
	AutoCap       bool   `toml:"auto_capitalize"`
	InjectDelayMs int    `toml:"inject_delay_ms"`
}

type TrayConfig struct {
	Enabled bool `toml:"enabled"`
}

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			URL:         "ws://localhost:5182/ws/firmware",
			InsecureTLS: true,
		},
		Audio: AudioConfig{
			Source:     "",
			SampleRate: 16000,
			ChunkMs:    100,
		},
		Mode: ModeConfig{
			Default: "ptt",
		},
		Hotkey: HotkeyConfig{
			PTTKey:    "KEY_SCROLLLOCK",
			ToggleKey: "KEY_PAUSE",
		},
		Inject: InjectConfig{
			Method:        "ydotool",
			TrailingSpace: true,
			AutoCap:       true,
			InjectDelayMs: 50,
		},
		Tray: TrayConfig{
			Enabled: true,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Defaults()

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, nil
		}
		path = filepath.Join(home, ".config", "local-stt-linux", "config.toml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
