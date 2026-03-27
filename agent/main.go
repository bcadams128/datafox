package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var logs = []string{"/var/log/apt/*.log", "/home/ben/logs/*.log"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	agent := NewAgent(logs, "offset.backup", "localhost:8080")

	if err := agent.Run(ctx); err != nil {
		log.Fatal(err)
	}

	log.Print("Shutdown complete")
}
