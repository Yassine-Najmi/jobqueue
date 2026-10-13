# Job Queue

A durable background job queue in Go, backed by PostgreSQL. Jobs are
submitted over HTTP and processed by a pool of concurrent workers.
Jobs survive crashes and restarts, and failed jobs are retried with
backoff.

Job types are pluggable handlers. The first real workload is syncing
ERP invoices (Sage, EBP) into Odoo, replacing a set of N8N workflows.

## Guarantees

| Guarantee | How | Test |
|---|---|---|
| Jobs survive a crash | Every state change is written to Postgres | TestPostgresStoreCreate |
| A job is never claimed twice | `FOR UPDATE SKIP LOCKED` in one atomic claim query | TestPostgresStoreClaimJob |
| Jobs held by a dead worker are recovered | Orphan reaper resets stale `running` jobs | TestPostgresStoreRecoverOrphanedJobs |
| Failures are retried, then marked failed | Attempt counter, `max_attempts`, `last_error` | TestPostgresStoreRecordAttempt, TestPostgresStoreMarkFailed |
| Shutdown loses no job and never panics | `context` is the only shutdown signal; channels are never closed | TestGracefulShutdown_NoPanicWithInFlightRetry |

Delivery is **at-least-once**: a job can run twice (for example after
a crash mid-handler), so handlers must be idempotent.
