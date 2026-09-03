package main

import (
	"log"
	"net/http"
)

func newRouter(store *InMemoryStore, registry map[string]JobHandler, jobChan chan<- Job) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /jobs", handleCreateJob(store, registry, jobChan))
	mux.HandleFunc("GET /jobs", handleListJobs(store))
	mux.HandleFunc("GET /jobs/{id}", handleGetJob(store))

	return loggingMiddleware(mux)
}

func main() {

	jobsChan := make(chan Job, 10)
	retryJobs := make(chan Job, 10)

	registry := map[string]JobHandler{"simulated": SimulatedHandler{}}

	store := NewInMemoryStore()

	startWorkerPool(3, jobsChan, retryJobs, store, registry)
	go retryDispatcher(retryJobs, jobsChan)

	router := newRouter(store, registry, jobsChan)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("server listening on :8080")

	err := server.ListenAndServe()
	log.Fatal(err)
}
