package main

import (
	"compress/gzip"
	"datafox/server/pkg/disk"
	"fmt"
	"io"
	"net/http"

	"github.com/vmihailenco/msgpack/v5"
)

type Server struct {
	ingestChan chan<- disk.LogBatch
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("POST /logs", s.handleLogs)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	fmt.Print("Pong")
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
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

	var batch disk.LogBatch
	if err := msgpack.NewDecoder(reader).Decode(&batch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case s.ingestChan <- batch:
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "server busy", http.StatusServiceUnavailable)
	}

}
