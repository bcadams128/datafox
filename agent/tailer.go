package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"syscall"
	"time"
)

type Tailer struct {
	path   string
	file   *os.File
	offset int64
	inode  uint64
	reader *bufio.Reader
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
		t.offset = 0
		t.inode = stat.Ino
	}

	if info.Size() > t.offset {
		for {
			line, err := t.reader.ReadString('\n')
			if len(line) > 0 {
				t.offset += int64(len(line))
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
