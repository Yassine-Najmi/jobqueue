package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func newTestPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()

	godotenv.Load()

	connString := fmt.Sprintf("postgres://%s:%s@localhost:5433/%s?sslmode=disable",
		os.Getenv("TEST_DB_USER"),
		os.Getenv("TEST_DB_PASSWORD"),
		os.Getenv("TEST_DB_NAME"),
	)

	store, err := NewPostgresStore(connString)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}

	if _, err := store.db.Exec("TRUNCATE TABLE jobs RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate jobs table: %v", err)
	}

	return store
}

func TestPostgresStoreCreate(t *testing.T) {
	store := newTestPostgresStore(t)

	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}

	job, err := store.Create(job)
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

func TestPostgresStoreGet(t *testing.T) {
	store := newTestPostgresStore(t)

	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}
	job, _ = store.Create(job)

	t.Run("existing job", func(t *testing.T) {
		getJob, err := store.Get(job.ID)

		if err != nil {
			t.Fatalf("expected no error, got : %v", err)
		}

		if getJob.Type != "email" {
			t.Fatalf("expected job type email, got : %v", getJob.Type)
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

func TestPostgresStoreMarkFailed(t *testing.T) {
	store := newTestPostgresStore(t)

	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}

	job, _ = store.Create(job)
	err := store.MarkFailed(job.ID)
	if err != nil {
		t.Fatalf("expected no error, got : %v", err)
	}
	job, _ = store.Get(job.ID)

	if job.Status != "failed" {
		t.Fatalf("expected status of the job failed, got status: %v", job.Status)
	}
}

func TestPostgresStoreRecordAttempt(t *testing.T) {
	store := newTestPostgresStore(t)

	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}
	job, _ = store.Create(job)
	job, err := store.RecordAttempt(job.ID)
	if err != nil {
		t.Fatalf("expected no error, got : %v", err)
	}

	if job.Attempts != 1 {
		t.Fatalf("expected attempts 1, got : %d", job.Attempts)
	}

	if job.Status != "retrying" {
		t.Fatalf("expected status retrying, got : %d", job.Attempts)
	}

	for attmpt := 1; attmpt <= job.MaxAttempts; attmpt++ {
		job, _ = store.RecordAttempt(job.ID)
	}

	if job.Attempts != job.MaxAttempts {
		t.Fatalf("expected attempts %d, got : %d", job.MaxAttempts, job.Attempts)
	}

	if job.Status != "failed" {
		t.Fatalf("expected status failed, got : %v", job.Status)
	}

}

func TestPostgresStoreClaimJob(t *testing.T) {
	store := newTestPostgresStore(t)

	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}

	job, _ = store.Create(job)
	job, err := store.ClaimJob("1")
	if err != nil {
		t.Fatalf("expected no error, got : %v", err)
	}

	if job.Status != "running" {
		t.Fatalf("expected status running, got : %v", job.Status)
	}

	if job.ClaimedBy != "1" {
		t.Fatalf("expected job claimed by 1, got : %v", job.ClaimedBy)
	}
}

func TestPostgresStoreecoverOrphanedJobs(t *testing.T) {
	store := newTestPostgresStore(t)

	jobs := make([]Job, 10)
	job := Job{
		Type:        "email",
		Payload:     map[string]any{"to": "user@example.com"},
		MaxAttempts: 3,
	}

	for i := 0; i < 10; i++ {
		j, _ := store.Create(job)
		j, _ = store.ClaimJob(strconv.Itoa(i))
		pastTime := time.Now().Add(-orphanTimeout - time.Second)
		store.db.Exec(`UPDATE jobs SET claimed_at = $1 WHERE id = $2`, pastTime, j.ID)
		jobs[i] = j
	}

	count, err := store.RecoverOrphanedJobs()
	if err != nil {
		t.Fatalf("expected no error, got : %v", err)
	}

	if count != 10 {
		t.Fatalf("expected count = 10, got : %v", count)
	}

	getAllJobs, _ := store.GetAll()

	for _, getJob := range getAllJobs {
		if getJob.Status != "retrying" {
			t.Fatalf("expected status retrying, got : %v", getJob.Status)
		}

		if getJob.Attempts != 1 {
			t.Fatalf("expected attempts 1, got : %d", getJob.Attempts)
		}
	}
}
