package disk

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type LogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

type DiskWriter struct {
	Dir       string
	current   *os.File
	timeStamp string
}

func (w *DiskWriter) WriteBatch(batch LogBatch) error {
	ts := time.Now().Format("20060102-1504")

	if w.current == nil || w.timeStamp != ts {
		if w.current != nil {
			w.current.Close()
		}

		path := filepath.Join(w.Dir, fmt.Sprintf("logs-%s.log", ts))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		w.current = f
		w.timeStamp = ts
	}

	for _, line := range batch.Lines {
		_, err := fmt.Fprintf(w.current, "%s [%s][%s] %s\n",
			time.Now().Format(time.RFC3339),
			batch.Metadata["hostname"],
			batch.Metadata["source"],
			line,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func DiskWorker(ch <-chan LogBatch, w *DiskWriter) {
	for batch := range ch {
		if err := w.WriteBatch(batch); err != nil {
			log.Printf("disk write failed: %v", err)
		}
	}
}
