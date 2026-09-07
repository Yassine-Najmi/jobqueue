# Job Queue

A background job processing service written in Go.

Jobs can be submitted over HTTP and are processed concurrently by a pool of worker goroutines. Each job can be tracked by its ID.

## Current Features

* HTTP API for submitting jobs
* In-memory job storage
* Worker pool using goroutines
* Concurrent job processing
* Job status tracking by ID
* Mutex-protected shared state
* Channels for job distribution
* Graceful shutdown
