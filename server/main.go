package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/vmihailenco/msgpack/v5"
)

type LogBatch struct {
	Lines    []string          `msgpack:"lines"`
	Metadata map[string]string `msgpack:"metadata"`
}

func main() {
	mux := http.NewServeMux()

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

		for _, line := range batch.Lines {
			log.Printf("[%s][%s] %s",
		        batch.Metadata["hostname"],
		        batch.Metadata["source"],
		        line,
			)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	log.Println("Starting Server")
	http.ListenAndServe("localhost:8080", mux)
}
