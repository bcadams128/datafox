package main

import (
	"datafox/server/pkg/disk"
	"log"
	"net/http"
	"time"
)

func main() {

	ingestChan := make(chan disk.LogBatch, 1000)
	srv := &Server{ingestChan: ingestChan}
	mux := http.NewServeMux()
	srv.routes(mux)

	writer := &disk.DiskWriter{
		Dir: "./logs",
		Now: time.Now,
	}

	go disk.DiskWorker(ingestChan, writer)

	log.Println("Starting Server")
	http.ListenAndServe("localhost:8080", mux)
}
