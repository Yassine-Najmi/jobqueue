package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrJobNotFound = errors.New("job not found")

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

func (s *InMemoryStore) Get(id int) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.jobs[id]

	if !exists {
		return Job{}, fmt.Errorf("get job %v : %w", id, ErrJobNotFound)
	}

	return job, nil
}

func (s *InMemoryStore) GetAll() ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *InMemoryStore) MarkRunning(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("mark running job %d: %w", id, ErrJobNotFound)
	}
	job.Status = "running"
	job.UpdatedAt = time.Now()

	s.jobs[id] = job

	return nil
}

func (s *InMemoryStore) MarkSuccess(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("mark success job %d: %w", id, ErrJobNotFound)
	}

	job.Status = "success"
	job.UpdatedAt = time.Now()

	s.jobs[id] = job

	return nil
}

func (s *InMemoryStore) MarkFailed(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("mark failed job %d: %w", id, ErrJobNotFound)
	}

	job.Status = "failed"
	job.UpdatedAt = time.Now()

	s.jobs[id] = job

	return nil
}

func (s *InMemoryStore) RecordAttempt(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("increment attempts job %d: %w", id, ErrJobNotFound)
	}

	if job.Attempts < job.MaxAttempts {
		job.Status = "retrying"
		job.Attempts++
	} else {
		job.Status = "failed"
	}

	job.UpdatedAt = time.Now()

	s.jobs[id] = job

	return nil
}

func main() {
	fmt.Println("Job Queue - work in progress")

	store := NewInMemoryStore()

	store.Create(Job{})
	store.Create(Job{})
	store.Create(Job{})
	job, _ := store.Create(Job{})

	fmt.Printf("the store id : %v the status is %s and the attmepts : %v", job.ID, job.Status, job.Attempts)

	// fmt.Println(store.Get(999))
	fmt.Println(store.GetAll())

}
