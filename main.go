package main

import (
	"fmt"
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
	jobs map[int]Job
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		jobs: make(map[int]Job),
	}
}

func main() {
	fmt.Println("Job Queue - work in progress")
}
