package main

import (
	"datafox/server/pkg/disk"
	"log"
	"net/http"
	"time"
)

func main() {

	ingestChan := make(chan disk.RawLogBatch, 1000)
	srv := &Server{ingestChan: ingestChan}
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
