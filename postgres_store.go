package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connString string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Create(job Job) (Job, error) {

	query := `INSERT INTO jobs (type, payload, status, attempts, max_attempts)
	VALUES ($1, $2, 'queued', 0, $3)
	RETURNING id, created_at, updated_at
	`
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("marshal payload error: %w", err)
	}

	newJob := job
	newJob.Status = "queued"
	newJob.Attempts = 0

	err = s.db.QueryRow(query, job.Type, payloadJSON, job.MaxAttempts).Scan(&newJob.ID, &newJob.CreatedAt, &newJob.UpdatedAt)
	if err != nil {
		return Job{}, fmt.Errorf("create job : %w", err)
	}

	return newJob, nil
}

func (s *PostgresStore) Get(id int) (Job, error) {
	var job Job
	var payloadJSON []byte
	query := `SELECT * FROM jobs
	WHERE id = $1
	`

	err := s.db.QueryRow(query).Scan(&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Attempts, &job.MaxAttempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, fmt.Errorf("get job %v : %w", id, ErrJobNotFound)
		}
		return Job{}, fmt.Errorf("get job %v : %w", id, err)
	}

	err = json.Unmarshal(payloadJSON, &job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("unmarshal payload error : %w", err)
	}

	return job, nil

}
