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
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/vmihailenco/msgpack/v5"
)

type Server struct {
	//parquet change
	ingestChan chan<- disk.RawLogBatch
	db         *sql.DB
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
	q := r.URL.Query()
	var where []string
	var args []any

	filters := []struct {
		param  string
		clause string
		like   bool
	}{
		{"source", "source ILIKE ?", true},
		{"host", "host = ?", false},
		{"message", "message ILIKE ?", true},
	}

	for _, f := range filters {
		value := q.Get(f.param)
		if value == "" {
			continue
		}
		if f.like {
			value = "%" + value + "%" // ILIKE needs wildcards for a substring match, not exact
		}
		where = append(where, f.clause)
		args = append(args, value)
	}

	start := time.Now()

	query := "SELECT timestamp, host, source, message FROM 'logs/*.parquet'"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY timestamp"

	rows, err := s.db.QueryContext(r.Context(), query, args...)
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

	elapsed := time.Since(start)
	log.Printf("query took %dms, %d rows", elapsed.Milliseconds(), len(results))
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
