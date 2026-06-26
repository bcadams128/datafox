package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
)

type tailerSupervisor struct {
	mu      sync.Mutex
	running map[string]*Tailer
	offsets *Offset
	out     chan LogLine
	rootCtx context.Context
	wg      sync.WaitGroup
}

type Tailer struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	offset int64
	inode  uint64
	reader *bufio.Reader
	cancel context.CancelFunc
}

func (super *tailerSupervisor) Reconcile(desired []string) {
	super.mu.Lock()
	defer super.mu.Unlock()

	want := make(map[string]bool, len(desired))
	for _, path := range desired {
		want[path] = true
	}
	for _, path := range desired {
		if _, ok := super.running[path]; !ok {
			super.start(path)
		}
	}
	for path, tailer := range super.running {
		if !want[path] {
			tailer.cancel()
			super.offsets.Files[path] = tailer.State()
			delete(super.running, path)
			log.Printf("stopped tailer for %s", path)
		}
	}
}

func (super *tailerSupervisor) start(path string) {
	tailer, err := NewLogTailer(path, super.offsets)
	if err != nil {
		log.Printf("tailer start failed for %s: %v", path, err)
		return
	}
	ctx, cancel := context.WithCancel(super.rootCtx)
	tailer.cancel = cancel
	super.running[path] = tailer

	super.wg.Add(1)
	go func() {
		defer super.wg.Done()
		if err := tailer.Poll(ctx, super.out); err != nil {
			log.Printf("tailer %s exited: %v", path, err)
		}
		super.mu.Lock()
		if super.running[path] == tailer {
			super.offsets.Files[path] = tailer.State()
			delete(super.running, path)
		}
		super.mu.Unlock()
	}()
	log.Printf("started tailer for %s", path)
}

func (super *tailerSupervisor) Offsets() *Offset {
	super.mu.Lock()
	defer super.mu.Unlock()
	offset := &Offset{Version: 1, Files: make(map[string]OffsetState)}
	for path, offsetState := range super.offsets.Files {
		offset.Files[path] = offsetState
	}
	for path, tailer := range super.running {
		offset.Files[path] = tailer.State()
	}
	return offset
}

func (t *Tailer) read(out chan<- LogLine) error {
	info, err := os.Stat(t.path)
	if err != nil {
		return err
	}

	stat := info.Sys().(*syscall.Stat_t)

	if stat.Ino != t.inode {
		t.file.Close()
		newFile, err := os.Open(t.path)
		if err != nil {
			return err
		}
		t.file = newFile
		t.reader = bufio.NewReader(newFile)
		t.mu.Lock()
		t.offset = 0
		t.inode = stat.Ino
		t.mu.Unlock()
	}

	if info.Size() > t.offset {
		for {
			line, err := t.reader.ReadString('\n')
			if len(line) > 0 {
				t.mu.Lock()
				t.offset += int64(len(line))
				t.mu.Unlock()
				out <- LogLine{Log: line, Source: t.path}
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func NewLogTailer(path string, savedOffsets *Offset) (*Tailer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	stat := info.Sys().(*syscall.Stat_t)
	currentInode := stat.Ino

	var offset int64 = 0
	if savedState, exists := savedOffsets.Files[path]; exists {
		if savedState.Inode == currentInode {
			offset = savedState.Offset
			if _, err := file.Seek(offset, 0); err != nil {
				return nil, err
			}
		}
	}

	return &Tailer{
		path:   path,
		file:   file,
		offset: offset,
		reader: bufio.NewReader(file),
		inode:  stat.Ino,
	}, nil
}

func (t *Tailer) Poll(ctx context.Context, out chan<- LogLine) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := t.read(out); err != nil {
				return err
			}
		}
	}
}

func (t *Tailer) State() OffsetState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return OffsetState{Path: t.path, Inode: t.inode, Offset: t.offset}
}
