package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const orphanTimeout = 30 * time.Second

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
	var claimedBy sql.NullString
	var payloadJSON []byte
	query := `SELECT id, type, payload, status, attempts, max_attempts, created_at, updated_at, claimed_at, claimed_by FROM jobs
	WHERE id = $1
	`

	err := s.db.QueryRow(query, id).Scan(&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt, &job.ClaimedAt, &claimedBy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, fmt.Errorf("get job %v : %w", id, ErrJobNotFound)
		}
		return Job{}, fmt.Errorf("get job %v : %w", id, err)
	}

	if claimedBy.Valid {
		job.ClaimedBy = claimedBy.String
	}

	err = json.Unmarshal(payloadJSON, &job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("unmarshal payload error : %w", err)
	}

	return job, nil

}

func (s *PostgresStore) GetAll() ([]Job, error) {
	jobs := make([]Job, 0)

	query := `SELECT id, type, payload, status, attempts, max_attempts, created_at, updated_at, claimed_at, claimed_by FROM jobs`

	rows, err := s.db.Query(query)
	if err != nil {
		return jobs, fmt.Errorf("get all jobs : %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var job Job
		var claimedBy sql.NullString
		var payloadJSON []byte

		if err := rows.Scan(&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt, &job.ClaimedAt, &claimedBy); err != nil {
			return jobs, fmt.Errorf("get all tasks : %w", err)
		}

		if claimedBy.Valid {
			job.ClaimedBy = claimedBy.String
		}

		err := json.Unmarshal(payloadJSON, &job.Payload)
		if err != nil {
			return jobs, fmt.Errorf("unmarshal payload error : %w", err)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *PostgresStore) MarkRunning(id int) error {

	query := `UPDATE jobs SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := s.db.Exec(query, "running", time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark running error : %w", err)
	}

	rowEffected, rowErr := result.RowsAffected()
	if rowErr != nil {
		return fmt.Errorf("row effected error : %w", rowErr)
	} else if rowEffected == 0 {
		return fmt.Errorf("update markRunning job %d : %w", id, ErrJobNotFound)
	}
	return nil
}

func (s *PostgresStore) MarkSuccess(id int) error {

	query := `UPDATE jobs SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := s.db.Exec(query, "success", time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark success error : %w", err)
	}

	rowEffected, rowErr := result.RowsAffected()
	if rowErr != nil {
		return fmt.Errorf("row effected error : %w", rowErr)
	} else if rowEffected == 0 {
		return fmt.Errorf("update markSuccess job %d : %w", id, ErrJobNotFound)
	}

	return nil
}

func (s *PostgresStore) MarkFailed(id int) error {
	query := `UPDATE jobs SET status = $1, updated_at = $2 WHERE ID = $3`

	result, err := s.db.Exec(query, "failed", time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark failed error : %w", err)
	}

	rowEffected, rowErr := result.RowsAffected()
	if rowErr != nil {
		return fmt.Errorf("row effected error : %w", rowErr)
	} else if rowEffected == 0 {
		return fmt.Errorf("update markFailed job %d : %w", id, ErrJobNotFound)
	}

	return nil
}

func (s *PostgresStore) RecordAttempt(id int) (Job, error) {
	var job Job
	var claimedBy sql.NullString
	var payloadJSON []byte

	query := `UPDATE jobs
	SET attempts = CASE
	WHEN attempts >= max_attempts THEN attempts
	ELSE attempts + 1
	END,
	status = CASE
	WHEN attempts >= max_attempts THEN 'failed'
	ELSE 'retrying'
	END,
	updated_at = $1,
	available_at = $3
	WHERE id = $2
	RETURNING *
	`
	err := s.db.QueryRow(query, time.Now(), id, time.Now().Add(500*time.Millisecond)).Scan(&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt, &job.ClaimedAt, &claimedBy, &job.AvailableAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, fmt.Errorf("record attempt job %d : %w", id, ErrJobNotFound)
		}
		return Job{}, fmt.Errorf("record attempt job %d : %w", id, err)
	}

	if claimedBy.Valid {
		job.ClaimedBy = claimedBy.String
	}

	err = json.Unmarshal(payloadJSON, &job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("unmarshal error : %w", err)
	}

	return job, nil
}

func (s *PostgresStore) ClaimJob(workerID string) (Job, error) {

	var job Job
	var claimedBy sql.NullString
	var payloadJSON []byte

	query := `
	UPDATE jobs SET
	status = 'running',
	updated_at = $1,
	claimed_at = $2,
	claimed_by = $3
	WHERE id = (
		SELECT id FROM jobs
		WHERE status IN ('queued', 'retrying')
		AND (available_at IS NULL OR available_at < now())
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED
		)
		RETURNING *
	`
	err := s.db.QueryRow(query, time.Now(), time.Now(), workerID).Scan(&job.ID, &job.Type, &payloadJSON, &job.Status, &job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt, &job.ClaimedAt, &claimedBy, &job.AvailableAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, ErrNoJobAvailable
		}
		return Job{}, fmt.Errorf("claim job error : %w", err)
	}

	if claimedBy.Valid {
		job.ClaimedBy = claimedBy.String
	}

	err = json.Unmarshal(payloadJSON, &job.Payload)
	if err != nil {
		return Job{}, fmt.Errorf("unmarshal error : %w", err)
	}

	return job, nil
}

func (s *PostgresStore) RecoverOrphanedJobs() (int, error) {

	query := `
	UPDATE jobs SET
	attempts = CASE
	WHEN attempts >= max_attempts THEN attempts
	ELSE attempts + 1
	END,
	status = CASE
	WHEN attempts >= max_attempts THEN 'failed'
	ELSE 'retrying'
	END,
	updated_at = $1
	WHERE  id IN (
		SELECT id FROM jobs
		WHERE status = 'running'
		AND claimed_at < $2
	)
	RETURNING id
	`
	rows, err := s.db.Query(query, time.Now(), time.Now().Add(-orphanTimeout))
	if err != nil {
		return 0, fmt.Errorf("Recover orphaned jobs error : %w", err)
	}

	defer rows.Close()

	var count int

	for rows.Next() {
		var id int

		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("recover a job error : %w", err)
		}
		count++

	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate recovered jobs: %w", err)
	}

	return count, nil
}
