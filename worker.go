package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func worker(ctx context.Context, id int, jobs <-chan Job, retryJobs chan<- Job, store Storage, registry map[string]JobHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-jobs:
			handler := registry[job.Type]

			if err := store.MarkRunning(job.ID); err != nil {
				fmt.Printf("worker %d: %v\n", id, err)
				continue
			}

			fmt.Printf("worker %d: processing job %d (%s)\n", id, job.ID, job.Type)

			err := handler.Handle(job)

			if err != nil {
				recordJob, recordErr := store.RecordAttempt(job.ID)
				if recordErr != nil {
					fmt.Printf("worker %d: %v\n", id, recordErr)
					continue
				}

				if recordJob.Status == "retrying" {
					fmt.Printf("worker %d: job %d failed, retrying (attempt %d/%d)\n", id, job.ID, recordJob.Attempts, recordJob.MaxAttempts)

					select {
					case <-time.After(500 * time.Millisecond):
					case <-ctx.Done():
						return
					}

					select {
					case retryJobs <- recordJob:
					case <-ctx.Done():
						return
					}
				} else {
					fmt.Printf("worker %d: job %d failed permanently\n", id, job.ID)
				}
			} else {
				store.MarkSuccess(job.ID)
				fmt.Printf("worker %d: job %d succeeded\n", id, job.ID)
			}
		}
	}
}

func startWorkerPool(ctx context.Context, numWorkers int, jobs chan Job, retryJobs chan Job, store Storage, registry map[string]JobHandler, workerWg *sync.WaitGroup) {
	workerWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer workerWg.Done()
			worker(ctx, workerID, jobs, retryJobs, store, registry)
		}(i)
	}
}
