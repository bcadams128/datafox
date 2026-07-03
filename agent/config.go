package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	LogPaths   []string `koanf:"log_paths"`
	Port       uint     `koanf:"port"`
	BatchSize  uint     `koanf:"batch_size"`
	Server     string   `koanf:"server"`
	OffsetPath string   `koanf:"offset_path"`
}

type ConfigStore struct {
	ptr atomic.Pointer[Config]
}

type Debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
	delay time.Duration
}

var ErrNoLogPaths = errors.New("config has no log_paths")

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{delay: delay}
}

func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, fn)
}

func (store *ConfigStore) Load() *Config { return store.ptr.Load() }

func parseConfig(path string) (*Config, error) {
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}

	var config Config
	if err := k.Unmarshal("agent", &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	log.Print(config.LogPaths)
	if len(config.LogPaths) == 0 {
		return nil, ErrNoLogPaths
	}
	return &config, nil
}

func NewConfig() (*ConfigStore, error) {
	cfg, err := parseConfig("config.toml")
	if err != nil {
		return nil, err
	}
	s := &ConfigStore{}
	s.ptr.Store(cfg)

	file := file.Provider("config.toml")
	log.Printf("Starting LogPaths: %s", cfg.LogPaths)

	debouncer := NewDebouncer(time.Second)
	file.Watch(func(event any, err error) {
		if err != nil {
			log.Printf("Watch error: %v", err)
			return
		}

		debouncer.Trigger(func() {
			next, err := parseConfig("config.toml")
			if err != nil {
				if errors.Is(err, ErrNoLogPaths) {
					log.Printf("no log_paths found keeping previous config")
					return
				}
				log.Printf("config reload failed, keeping previous config :%v", err)
				return
			}

			log.Println("New config found reloading")
			s.ptr.Store(next)
		})
	})
	return s, nil
}
