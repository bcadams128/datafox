package main

import (
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	agent := NewAgent(config)

	if err := agent.Run(ctx); err != nil {
		log.Fatal(err)
	}

	log.Print("Shutdown complete")
}
