package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan Job, retryJobs chan<- Job, store *InMemoryStore, registry map[string]JobHandler) {
	for job := range jobs {
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
				time.Sleep(500 * time.Millisecond)
				retryJobs <- recordJob
			} else {
				fmt.Printf("worker %d: job %d failed permanently\n", id, job.ID)
			}
		} else {
			store.MarkSuccess(job.ID)
			fmt.Printf("worker %d: job %d succeeded\n", id, job.ID)
		}
	}
}

func startWorkerPool(numWorkers int, jobs chan Job, retryJobs chan Job, store *InMemoryStore, registry map[string]JobHandler, wg *sync.WaitGroup) {
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			worker(workerID, jobs, retryJobs, store, registry)
		}(i)
	}
}
