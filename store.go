package main

import (
	"errors"
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
	ClaimedAt   *time.Time     `json:"claimed_at,omitempty"`
	ClaimedBy   string         `json:"claimed_by,omitempty"`
}

type Storage interface {
	Create(job Job) (Job, error)
	Get(id int) (Job, error)
	GetAll() ([]Job, error)
	MarkRunning(id int) error
	MarkSuccess(id int) error
	RecordAttempt(id int) (Job, error)
}
