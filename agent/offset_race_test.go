package main

import (
	"os"
	"sync"
	"testing"
)

func TestTailerOffsetRace(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "log")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tl, err := NewLogTailer(f.Name(), &Offset{Files: map[string]OffsetState{}})
	if err != nil {
		t.Fatal(err)
	}

	out := make(chan LogLine, 1000)
	go func() {
		for range out {
		}
	}() // drain

	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // writer: read() does t.offset += len(line)
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			f.WriteString("line\n")
			tl.read(out)
		}
	}()

	go func() { // reader: the saver path reads t.offset / t.inode
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = TailersToOffsets([]*Tailer{tl})
		}
	}()

	wg.Wait()
}
