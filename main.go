package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Job Queue - work in progress")

	jobsChan := make(chan Job, 10)

	registry := map[string]JobHandler{"simulated": SimulatedHandler{}}

	store := NewInMemoryStore()

	startWorkerPool(3, jobsChan, jobsChan, store, registry)

	for i := 0; i < 5; i++ {
		job, err := store.Create(Job{Type: "simulated", MaxAttempts: 3})
		if err != nil {
			fmt.Printf("job creation error: %v\n", err)
			continue
		}
		jobsChan <- job
	}

	time.Sleep(5 * time.Second)
}
