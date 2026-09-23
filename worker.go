package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

func worker(ctx context.Context, workerID int, jobs <-chan Job, retryJobs chan<- Job, store Storage, registry map[string]JobHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-jobs:
			handler, ok := registry[job.Type]
			if !ok {
				log.Printf("key doesn't exist")
				continue
			}

			if err := store.MarkRunning(job.ID); err != nil {
				fmt.Printf("worker %d: %v\n", workerID, err)
				continue
			}

			fmt.Printf("worker %d: processing job %d (%s)\n", workerID, job.ID, job.Type)

			err := handler.Handle(job)

			if err != nil {
				recordJob, recordErr := store.RecordAttempt(job.ID)
				if recordErr != nil {
					fmt.Printf("worker %d: %v\n", workerID, recordErr)
					continue
				}

				if recordJob.Status == "retrying" {
					fmt.Printf("worker %d: job %d failed, retrying (attempt %d/%d)\n", workerID, job.ID, recordJob.Attempts, recordJob.MaxAttempts)

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
					fmt.Printf("worker %d: job %d failed permanently\n", workerID, job.ID)
				}
			} else {
				store.MarkSuccess(job.ID)
				fmt.Printf("worker %d: job %d succeeded\n", workerID, job.ID)
			}
		}
	}
}

func pgWorker(ctx context.Context, workerID int, store *PostgresStore, registry map[string]JobHandler) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job, err := store.ClaimJob(strconv.Itoa(workerID))
			if err != nil {
				if errors.Is(err, ErrNoJobAvailable) {
					continue
				} else {
					fmt.Printf("job claim, worker %d: %v\n", workerID, err)
					continue
				}
			}

			fmt.Printf("worker %d: processing job %d (%s)\n", workerID, job.ID, job.Type)

			handler, ok := registry[job.Type]
			if !ok {
				log.Printf("key doesn't exist")
				err := store.MarkFailed(job.ID)
				if err != nil {
					if errors.Is(err, ErrJobNotFound) {
						continue
					} else {
						fmt.Printf("job mark failed error %d: %v\n", workerID, err)
						continue
					}
				}
				continue
			}

			err = handler.Handle(job)

			if err != nil {
				recordJob, recordErr := store.RecordAttempt(job.ID)
				if recordErr != nil {
					fmt.Printf("worker %d: %v\n", workerID, recordErr)
					continue
				}

				if recordJob.Status == "retrying" {
					fmt.Printf("worker %d: job %d failed, retrying (attempt %d/%d)\n", workerID, job.ID, recordJob.Attempts, recordJob.MaxAttempts)
				} else {
					fmt.Printf("worker %d: job %d failed permanently\n", workerID, job.ID)
				}
			} else {
				store.MarkSuccess(job.ID)
				fmt.Printf("worker %d: job %d succeeded\n", workerID, job.ID)
			}
		}
	}
}

func startWorkerPool(ctx context.Context, numWorkers int, jobs chan Job, retryJobs chan Job, store Storage, registry map[string]JobHandler, workerWg *sync.WaitGroup) {
	workerWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer workerWg.Done()
			if pgStore, ok := store.(*PostgresStore); ok {
				pgWorker(ctx, workerID, pgStore, registry)
			} else {
				worker(ctx, workerID, jobs, retryJobs, store, registry)
			}
		}(i)
	}
}
