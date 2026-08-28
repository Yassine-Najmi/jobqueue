package main

import (
	"errors"
	"testing"
)

func TestInMemoryStoreCreate(t *testing.T) {
	store := NewInMemoryStore()

	// job, err := store.Create(Job{Type: "send_email", Status: "queued"})
	job, err := store.Create(Job{Type: "send_email"})

	if err != nil {
		t.Fatalf("expected no error, got : %v", err)
	}

	if job.ID != 1 {
		t.Fatalf("expected first job ID to be 1 got %v", job.ID)
	}

	if job.Status != "queued" || job.Attempts != 0 {
		t.Fatalf("expected status of the job queued and attempts 0, got status: %v and attempts: %v", job.Status, job.Attempts)
	}
}

func TestInMemoryStoreGet(t *testing.T) {
	store := NewInMemoryStore()

	job, _ := store.Create(Job{Type: "send_email"})

	t.Run("existing job", func(t *testing.T) {
		getJob, err := store.Get(job.ID)

		if err != nil {
			t.Fatalf("expected no error, got : %v", err)
		}

		if getJob.Type != "send_email" {
			t.Fatalf("expected job type send_email, got : %v", getJob.Type)
		}
	})

	t.Run("nonexistent job", func(t *testing.T) {
		_, err := store.Get(999)

		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("expected error job not found, got : %v", err)
		}
	})
}
