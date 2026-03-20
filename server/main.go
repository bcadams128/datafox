package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type LogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

type DiskWriter struct {
	dir       string
	current   *os.File
	timeStamp string
}

func main() {
	mux := http.NewServeMux()
	ingestChan := make(chan LogBatch, 1000)

	writer := &DiskWriter{
		dir: "./logs",
	}

	go diskWorker(ingestChan, writer)

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Print("Pong")
	})

	mux.HandleFunc("POST /logs", func(w http.ResponseWriter, r *http.Request) {
		var reader io.Reader = r.Body
		defer r.Body.Close()

		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			reader = gz
		}

		var batch LogBatch
		if err := msgpack.NewDecoder(reader).Decode(&batch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		select {
		case ingestChan <- batch:
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "server busy", http.StatusServiceUnavailable)
		}

	})

	log.Println("Starting Server")
	http.ListenAndServe("localhost:8080", mux)
}

func (w *DiskWriter) writeBatch(batch LogBatch) error {
	ts := time.Now().Format("20060102-1504")

	if w.current == nil || w.timeStamp != ts {
		if w.current != nil {
			w.current.Close()
		}

		path := filepath.Join(w.dir, fmt.Sprintf("logs-%s.log", ts))
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

func diskWorker(ch <-chan LogBatch, w *DiskWriter) {
	for batch := range ch {
		if err := w.writeBatch(batch); err != nil {
			log.Printf("disk write failed: %v", err)
		}
	}
}
