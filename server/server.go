package main

import (
	"compress/gzip"
	"datafox/server/pkg/disk"
	"io"
	"log"
	"net/http"

	"github.com/vmihailenco/msgpack/v5"
)

type Server struct {
	//parquet change
	ingestChan chan<- disk.RawLogBatch
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("POST /logs", s.handleLogs)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pong"))
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	log.Printf("log endpoint hit")
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
	//Reject if not gzip

	var batch disk.RawLogBatch
	if err := msgpack.NewDecoder(reader).Decode(&batch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Log Batch sent to channel")

	select {
	case s.ingestChan <- batch:
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "server busy", http.StatusServiceUnavailable)
	}

}
