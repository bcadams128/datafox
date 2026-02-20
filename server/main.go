package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Print("Pong")
	})

	mux.HandleFunc("POST /logs", func(w http.ResponseWriter, r *http.Request) {
        defer r.Body.Close()

		scanner := bufio.NewScanner(r.Body)
		for scanner.Scan() {
			log.Printf("[log] %s", scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})


	log.Println("Starting Server")
	http.ListenAndServe("localhost:8080", mux)
}
