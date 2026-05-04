package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"handling_large_files_go/internal/db"
	"handling_large_files_go/internal/upload"
	"handling_large_files_go/internal/worker"
)

func main() {
	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}

	// connect to DB
	if err := db.Connect(); err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.EnsureMigration(); err != nil {
		log.Fatalf("db migration: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/upload", upload.UploadHandler).Methods("POST")
	r.HandleFunc("/status/{job_id}", upload.StatusHandler).Methods("GET")

	// start workers in background
	go worker.StartWorkers()

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
