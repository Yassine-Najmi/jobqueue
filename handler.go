package main

import (
	"fmt"
	"math/rand"
	"time"
)

// JobHandler is implemented once per job type. Handle only performs the
// actual work, it never manages job lifecycle (status, attempts); that
// stays the worker's responsibility.
type JobHandler interface {
	Handle(job Job) error
}

// SimulatedHandler stands in for real work (sending an email, generating a PDF, etc.)
// while the worker pool/retry/concurrency machinery is being
// built and tested, without needing any real external dependency.
type SimulatedHandler struct{}

func (h SimulatedHandler) Handle(job Job) error {
	time.Sleep(500 * time.Millisecond)

	if rand.Intn(2) == 0 {
		return fmt.Errorf("the simulator failed")
	}
	return nil
}
