package main

import "testing"

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
