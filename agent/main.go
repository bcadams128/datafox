package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg := config.Load()
	var logs = cfg.LogPaths
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	serverUrl := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	agent := NewAgent(logs, cfg.OffsetPath, serverUrl)

	if err := agent.Run(ctx); err != nil {
		log.Fatal(err)
	}

	log.Print("Shutdown complete")
}
