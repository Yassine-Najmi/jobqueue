package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
)

func newRouter(store Storage, registry map[string]JobHandler, jobChan chan<- Job) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /jobs", handleCreateJob(store, registry, jobChan))
	mux.HandleFunc("GET /jobs", handleListJobs(store))
	mux.HandleFunc("GET /jobs/{id}", handleGetJob(store))

	return loggingMiddleware(mux)
}

func newStore() (Storage, error) {
	godotenv.Load()

	if os.Getenv("STORE_TYPE") != "postgres" {
		return NewInMemoryStore(), nil
	}

	connString := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	return NewPostgresStore(connString)
}

func main() {
	var workerWg sync.WaitGroup
	var dispatcherWg sync.WaitGroup

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobsChan := make(chan Job, 10)
	retryJobs := make(chan Job, 10)

	registry := map[string]JobHandler{"simulated": SimulatedHandler{}}

	store, err := newStore()
	if err != nil {
		log.Fatal(err)
	}

	if pgStore, ok := store.(*PostgresStore); ok {
		count, err := pgStore.RecoverOrphanedJobs()
		if err != nil {
			log.Printf("Failed to recover orphaned jobs : %v", err)
		} else {
			log.Printf("Recovered %d orphaned jobs", count)
		}
	}

	startWorkerPool(ctx, 3, jobsChan, retryJobs, store, registry, &workerWg)

	dispatcherWg.Add(1)
	go retryDispatcher(ctx, retryJobs, jobsChan, &dispatcherWg)

	router := newRouter(store, registry, jobsChan)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutdown signal received, stopping server")
		server.Shutdown(context.Background())
	}()

	log.Println("server listening on :8080")
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println("server error:", err)
	}

	log.Println("waiting for in-flight jobs to finish...")
	workerWg.Wait()
	dispatcherWg.Wait()
	log.Println("shutdown complete")
}
