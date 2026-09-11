package main

import (
	"compress/gzip"
	"database/sql"
	"datafox/server/pkg/disk"
	"datafox/server/pkg/types"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/vmihailenco/msgpack/v5"
)

type Server struct {
	//parquet change
	ingestChan chan<- disk.RawLogBatch
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("POST /logs", s.handleLogs)
	mux.HandleFunc("GET /logs", s.readLogs)
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

func (s *Server) readLogs(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	q := r.URL.Query()
	var where []string
	var args []any

	if v := q.Get("source"); v != "" {
		where = append(where, "source ILIKE ?")
		args = append(args, "%"+v+"%")
	}
	if v := q.Get("host"); v != "" {
		where = append(where, "host = ?")
		args = append(args, v)
	}
	if v := q.Get("message"); v != "" {
		where = append(where, "message ILIKE ?")
		args = append(args, "%"+v+"%")
	}

	query := "SELECT timestamp, host, source, message FROM 'logs/*.parquet'"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY timestamp"

	rows, err := db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []types.LogRecord
	for rows.Next() {
		var rec types.LogRecord
		if err := rows.Scan(&rec.Timestamp, &rec.Host, &rec.Source, &rec.Message); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
