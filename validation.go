package main

import "errors"

func validateJob(job Job, registry map[string]JobHandler) []error {
	var errs []error

	_, exists := registry[job.Type]

	if !exists {
		errs = append(errs, errors.New("the job type isn't available"))
	}

	if job.MaxAttempts <= 0 {
		errs = append(errs, errors.New("the max attempts should be higher than 0"))
	}

	return errs
}
