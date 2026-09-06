package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// alwaysFailHandler deterministically fails every time, guaranteeing a
// retry is triggered on every run
type alwaysFailHandler struct{}

func (h alwaysFailHandler) Handle(job Job) error {
	time.Sleep(50 * time.Millisecond)
	return errors.New("always fails")
}

// TestGracefulShutdown_NoPanicWithInFlightRetry recreates the exact race
// condition manual Ctrl+C testing found: a job failing and being retried
// at the same moment shutdown begins. It's run many times, since this
// class of bug is timing-dependent and can pass by coincidence on any
// single run.
func TestGracefulShutdown_NoPanicWithInFlightRetry(t *testing.T) {
	for i := 0; i < 10; i++ {
		var workerWg sync.WaitGroup
		var dispatcherWg sync.WaitGroup

		ctx, cancel := context.WithCancel(context.Background())

		jobsChan := make(chan Job, 10)
		retryChan := make(chan Job, 10)
		registry := map[string]JobHandler{"fail": alwaysFailHandler{}}
		store := NewInMemoryStore()

		startWorkerPool(ctx, 2, jobsChan, retryChan, store, registry, &workerWg)

		dispatcherWg.Add(1)
		go retryDispatcher(ctx, retryChan, jobsChan, &dispatcherWg)

		job, _ := store.Create(Job{Type: "fail", MaxAttempts: 5})
		jobsChan <- job

		// Give a worker time to pick up the job and be mid-Handle() before
		// shutdown starts
		time.Sleep(10 * time.Millisecond)

		cancel()
		workerWg.Wait()
		dispatcherWg.Wait()
	}
}
