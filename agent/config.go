package main

import (
	"fmt"
	"log"
	"sync/atomic"

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
	file.Watch(func(event any, err error) {
		if err != nil {
			log.Printf("Watch error: %v", err)
			return
		}
		next, err := parseConfig("config.toml")
		if err != nil {
			log.Printf("config reload failed, keeping previous config :%v", err)
			return
		}
		log.Println("Config reloading")
		s.ptr.Store(next)
	})
	return s, nil
}
