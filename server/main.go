package main

import (
	"database/sql"
	"datafox/server/pkg/disk"
	"log"
	"net/http"
	"time"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ingestChan := make(chan disk.RawLogBatch, 1000)
	srv := &Server{ingestChan: ingestChan, db: db}
	mux := http.NewServeMux()
	srv.routes(mux)

	//parquet change
	writer := &disk.DiskWriter{
		Dir: "./logs",
		Now: time.Now,
	}

	//parquet change
	go disk.DiskWorker(ingestChan, writer)

	log.Println("Starting Server")
	http.ListenAndServe("localhost:8080", mux)
}
