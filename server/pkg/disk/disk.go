package disk

import (
	"context"
	"datafox/server/pkg/types"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/parquet-go/parquet-go"
)

type RawLogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

type DiskWriter struct {
	Dir       string
	current   *os.File
	writer    *parquet.GenericWriter[types.LogRecord]
	timeStamp string
	Now       func() time.Time
}

type LogStore interface {
	Write(ctx context.Context, logs []types.LogEntry) error
}

// Figure out where canonical timestamp SHOULD come from
func RawToEntry(batch RawLogBatch) []types.LogEntry {
	log.Printf("RawToEntry attempt")
	entries := make([]types.LogEntry, len(batch.Lines))
	for i, line := range batch.Lines {
		entries[i] = types.LogEntry{
			Timestamp: time.Now(),
			Host:      batch.Metadata["hostname"],
			Source:    batch.Metadata["source"],
			Message:   line,
		}
	}
	return entries
}

func (w *DiskWriter) WriteBatch(logs []types.LogEntry) error {
	ts := w.Now().Truncate(5 * time.Second).Format("20060102-150405")
	log.Printf("Starting new batch log")

	if w.current == nil || w.timeStamp != ts {
		if w.current != nil {
			if err := w.writer.Close(); err != nil {
				return err
			}

			if err := w.current.Close(); err != nil {
				return err
			}
		}

		path := filepath.Join(w.Dir, fmt.Sprintf("logs-%s.parquet", ts))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		w.current = f
		w.writer = parquet.NewGenericWriter[types.LogRecord](f)
		w.timeStamp = ts
	}
	logBatch := make([]types.LogRecord, 0, 100_000)
	for _, log := range logs {
		logBatch = append(logBatch, types.ToLogRecord(log))
	}
	if _, err := w.writer.Write(logBatch); err != nil {
		return err
	}
	return nil
}

func DiskWorker(ch <-chan RawLogBatch, w *DiskWriter) {
	log.Printf("writing to disk")
	log.Print(ch)
	for batch := range ch {
		log.Printf("Attempting to call RawToEntry")
		entries := RawToEntry(batch)
		if err := w.WriteBatch(entries); err != nil {
			log.Printf("disk write failed: %v", err)
		}
	}
}
