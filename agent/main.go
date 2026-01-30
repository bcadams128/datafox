package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type Tailer struct {
	path   string
	file   *os.File
	offset int64
	inode  uint64
	reader *bufio.Reader
}

type OffsetState struct {
	Path   string `msgpack:"Path"`
	Inode  uint64 `msgpack:"inode"`
	Offset int64  `msgpack:"Offset"`
}

type Offset struct {
	Files map[string]OffsetState
}

func main() {
	var logs = []string{"/var/log/apt/*.log", "/home/ben/logs/*.log"}
	paths, _ := discover(logs)
	var wg sync.WaitGroup
	ctx := context.Background()
	out := make(chan string, 1000)

	var tailers []*Tailer

	for _, path := range paths {
		t, err := NewLogTailer(path)
		if err != nil {
			panic(err)
		}

		tailers = append(tailers, t)

		wg.Add(1)
		go func(t *Tailer) {
			defer wg.Done()
			_ = t.Poll(ctx, out)
		}(t)

	}

	go func() {
		wg.Wait()
		close(out)
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				SaveOffsets("offset.backup", TailersToOffsets(tailers))
			}

		}
	}()

	for line := range out {
		print(line) // line already has \n from ReadString
	}
}

func (t *Tailer) read(out chan<- string) error {
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
		t.offset = 0
		t.inode = stat.Ino
	}

	if info.Size() > t.offset {
		for {
			line, err := t.reader.ReadString('\n')
			if len(line) > 0 {
				t.offset += int64(len(line))
				out <- line
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

func NewLogTailer(path string) (*Tailer, error) {
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

	return &Tailer{
		path:   path,
		file:   file,
		offset: 0,
		reader: bufio.NewReader(file),
		inode:  stat.Ino,
	}, nil
}

func (t *Tailer) Poll(ctx context.Context, out chan<- string) error {
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

func discover(globs []string) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string

	for _, g := range globs {
		matches, err := filepath.Glob(g)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			files = append(files, m)
		}
	}
	return files, nil
}

func SaveOffsets(path string, db *Offset) error {
	tmp := path + ".tmp"

	b, err := msgpack.Marshal(db)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}


func TailersToOffsets(tailers []*Tailer) *Offset {
	o := &Offset{Files: make(map[string]OffsetState)}
	for _, t := range tailers {
		o.Files[t.path] = OffsetState{
			Path:   t.path,
			Inode:  t.inode,
			Offset: t.offset,
		}
	}
	return o
}
