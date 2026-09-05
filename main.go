package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func newRouter(store *InMemoryStore, registry map[string]JobHandler, jobChan chan<- Job) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /jobs", handleCreateJob(store, registry, jobChan))
	mux.HandleFunc("GET /jobs", handleListJobs(store))
	mux.HandleFunc("GET /jobs/{id}", handleGetJob(store))

	return loggingMiddleware(mux)
}

func main() {
	var workerWg sync.WaitGroup
	var dispatcherWg sync.WaitGroup

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobsChan := make(chan Job, 10)
	retryJobs := make(chan Job, 10)
	shutdownDone := make(chan struct{})

	registry := map[string]JobHandler{"simulated": SimulatedHandler{}}

	store := NewInMemoryStore()

	startWorkerPool(3, jobsChan, retryJobs, store, registry, &workerWg)
	func() {
		dispatcherWg.Add(1)
		go retryDispatcher(ctx, retryJobs, jobsChan, &dispatcherWg)
	}()

	router := newRouter(store, registry, jobsChan)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutdown signal received, stopping server")
		server.Shutdown(context.Background())
		close(jobsChan)
		workerWg.Wait()
		close(retryJobs)
		dispatcherWg.Wait()
		close(shutdownDone)
	}()

	log.Println("server listening on :8080")
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println("server error:", err)
	}

	log.Println("waiting for in-flight jobs to finish...")
	<-shutdownDone
	log.Println("shutdown complete")
}
