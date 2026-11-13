package disk

import (
	"context"
	"datafox/server/pkg/types"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

type RawLogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

type DiskWriter struct {
	Dir     string
	current *os.File
	writer  *parquet.GenericWriter[types.LogRecord]
	Now     func() time.Time
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

	if w.current == nil {
		ts := w.Now().UTC().Format("20060102-150405.000")
		path := filepath.Join(w.Dir, fmt.Sprintf("logs-%s.parquet.tmp", ts))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)

		if err != nil {
			return err
		}

		w.current = f
		w.writer = parquet.NewGenericWriter[types.LogRecord](f)
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

func (w *DiskWriter) rotate() error {
	if w.current == nil {
		return nil
	}

	path := w.current.Name()
	writerErr := w.writer.Close()
	fileErr := w.current.Close()
	w.current = nil
	w.writer = nil

	if err := errors.Join(writerErr, fileErr); err != nil {
		return err
	}
	return os.Rename(path, strings.TrimSuffix(path, ".tmp"))
}

func DiskWorker(ch <-chan RawLogBatch, w *DiskWriter) {
	log.Printf("writing to disk")
	log.Print(ch)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case batch, ok := <-ch:
			if !ok {
				// Channel closes final rotate / close
				if err := w.rotate(); err != nil {
					log.Printf("final rotate failed: %v", err)
				}
				return
			}
			// This is the OK path
			entries := RawToEntry(batch)
			if err := w.WriteBatch(entries); err != nil {
				log.Printf("disk write failed: %v", err)
			}
		case <-ticker.C:
			log.Printf("Log file rotated")
			if err := w.rotate(); err != nil {
				log.Printf("rotate failed: %v", err)
			}
		}

	}
}

func Read(filename string, source string) ([]types.LogRecord, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	schema := parquet.SchemaOf(types.LogRecord{})
	sourceCol, _ := schema.Lookup("source")

	reader := parquet.NewReader(f, schema)
	filtered := parquet.FilterRowReader(reader, func(row parquet.Row) bool {
		return strings.Contains(row[sourceCol.ColumnIndex].String(), source)
	})

	var matches []types.LogRecord
	buf := make([]parquet.Row, 20)

	for {
		n, err := filtered.ReadRows(buf)
		for _, row := range buf[:n] {
			var rec types.LogRecord
			if rerr := schema.Reconstruct(&rec, row); rerr != nil {
				return nil, rerr
			}
			matches = append(matches, rec)
		}
		if err == io.EOF {

			break
		}
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}
