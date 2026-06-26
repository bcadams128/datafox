package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Agent struct {
	cfg        *ConfigStore
	offsetPath string
}

func NewAgent(cfg *ConfigStore) *Agent {
	return &Agent{
		cfg:        cfg,
		offsetPath: cfg.Load().OffsetPath,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	var batch []LogLine
	batchticker := time.NewTicker(5 * time.Second)
	reconcileticker := time.NewTicker(2 * time.Second)
	defer batchticker.Stop()

	fmt.Println("Starting Agent")
	paths, _ := discover(a.cfg.Load().LogPaths)

	out := make(chan LogLine, 1000)
	log.Print("Offset path ", a.offsetPath)
	savedOffsets, err := LoadOffsets(a.offsetPath)
	if err != nil {
		log.Fatal(err)
	}

	super := &tailerSupervisor{
		mu:      sync.Mutex{},
		offsets: savedOffsets,
		running: map[string]*Tailer{},
		rootCtx: ctx,
		out:     out,
		wg:      sync.WaitGroup{},
	}
	super.Reconcile(paths)

	go func() {
		offsetTicker := time.NewTicker(2 * time.Second)
		defer offsetTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-offsetTicker.C:
				SaveOffsets(a.offsetPath, super.Offsets())
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
		case <-reconcileticker.C:
			paths, _ = discover(a.cfg.Load().LogPaths)
			super.Reconcile(paths)
		case <-ctx.Done():
			SaveOffsets(a.offsetPath, super.Offsets())
			log.Print("context cancelled, exiting batch loop")
			return nil
		}
	}

}
