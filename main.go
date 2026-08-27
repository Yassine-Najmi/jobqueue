package main

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID          int            `json:"id"`
	Type        string         `json:"type"`
	Payload     map[string]any `json:"payload"`
	Status      string         `json:"status"` // queued, running, retrying, success, failed
	Attempts    int            `json:"attempts"`
	MaxAttempts int            `json:"max_attempts"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type InMemoryStore struct {
	jobs   map[int]Job
	nextID int
	mu     sync.Mutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		jobs: make(map[int]Job),
	}
}

func (s *InMemoryStore) generateID() int {
	s.nextID++
	return s.nextID
}

func (s *InMemoryStore) Create(job Job) (Job, error) {

	s.mu.Lock()
	defer s.mu.Unlock()
	timeNow := time.Now()
	jobID := s.generateID()
	s.jobs[jobID] = Job{
		ID:          jobID,
		Type:        job.Type,
		Payload:     job.Payload,
		Status:      "queued",
		Attempts:    0,
		MaxAttempts: job.MaxAttempts,
		CreatedAt:   timeNow,
		UpdatedAt:   timeNow,
	}

	return s.jobs[jobID], nil
}

func main() {
	fmt.Println("Job Queue - work in progress")

	store := NewInMemoryStore()

	store.Create(Job{})
	store.Create(Job{})
	store.Create(Job{})
	job, _ := store.Create(Job{})

	fmt.Printf("the store id : %v the status is %s and the attmepts : %v", job.ID, job.Status, job.Attempts)

}
