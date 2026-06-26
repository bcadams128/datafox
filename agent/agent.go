package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Agent struct {
	logPaths   []string
	offsetPath string
	serverUrl  string
}

func NewAgent(logPaths []string, offsetPath string, serverURL string) *Agent {
	return &Agent{
		logPaths:   logPaths,
		offsetPath: offsetPath,
		serverUrl:  serverURL,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	var batch []LogLine
	batchticker := time.NewTicker(5 * time.Second)
	defer batchticker.Stop()

	fmt.Println("Starting Agent")
	paths, _ := discover(a.logPaths)

	out := make(chan LogLine, 1000)
	log.Print("Offset path ", a.offsetPath)
	savedOffsets, err := LoadOffsets(a.offsetPath)
	if err != nil {
		log.Fatal(err)
	}

	var tailers []*Tailer

	for _, path := range paths {
		log.Print("Gettings logs from ", path)
		t, err := NewLogTailer(path, savedOffsets)
		if err != nil {
			panic(err)
		}

		tailers = append(tailers, t)

		wg.Add(1)
		go func(t *Tailer) {
			defer wg.Done()
			if err := t.Poll(ctx, out); err != nil {
				log.Printf("tailer %s exited with error: %v", t.path, err)
			}
		}(t)

	}

	go func() {
		wg.Wait()
		close(out)
	}()

	go func() {
		offsetTicker := time.NewTicker(2 * time.Second)
		defer offsetTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-offsetTicker.C:
				SaveOffsets(a.offsetPath, TailersToOffsets(tailers))
			}

		}
	}()

	for {
		select {
		case line, ok := <-out:
			if !ok {
				if len(batch) > 0 {
					a.sendWithRetry(batch)
				}
				return nil
			}
			batch = append(batch, line)
			if len(batch) >= 10 {
				a.sendWithRetry(batch)
				batch = nil
			}
		case <-batchticker.C:
			if len(batch) > 0 {
				a.sendWithRetry(batch)
				batch = nil
			}
		case <-ctx.Done():
			SaveOffsets(a.offsetPath, TailersToOffsets(tailers))
			log.Print("context cancelled, exiting batch loop")
			return nil
		}
	}

}
