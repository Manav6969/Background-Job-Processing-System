# Understanding the Background Job Processing System

This project is a distributed background job processing system built with **Go**, **PostgreSQL**, and **Redis**. It follows a producer-consumer architecture to handle long-running or asynchronous tasks efficiently.

---

## Core Components

The system is divided into several key components:

1. **API Service (`cmd/api`)**:
   - Acts as the **Producer**.
   - Exposes RESTful endpoints (using the Gin framework) for user authentication and job submission.
   - When a job is submitted, it:
     1. Validates the payload (type, required fields, and size limit — reject payloads above a configurable threshold, e.g., 1MB).
     2. Persists the job details in the **PostgreSQL** database with a `pending` status, assigning a unique **idempotency key**.
     3. Pushes the job metadata into a **Redis** queue for asynchronous processing.

2. **Worker Service (`cmd/worker`)**:
   - Acts as the **Consumer**.
   - Runs as a separate background process with a configurable **goroutine pool** for concurrent job processing (controlled via `WORKER_CONCURRENCY` env var).
   - Continuously monitors the **Redis** queue for new jobs using `BRPOPLPUSH` into a processing queue (prevents job loss on crash).
   - When a job is picked up:
     1. Checks the **idempotency key** to avoid duplicate execution.
     2. Updates the job status in the database to `running`, recording `started_at`.
     3. Executes the business logic within a `context.WithTimeout` to prevent runaway jobs.
     4. On success: updates the final status to `completed` with `finished_at`.
     5. On failure: applies **exponential backoff retry** (up to `MAX_RETRIES`). After exhausting retries, moves the job to the **Dead Letter Queue (DLQ)** and records the `error_message`.
   - Handles `SIGTERM`/`SIGINT` for **graceful shutdown** — drains in-flight jobs before exiting.

3. **PostgreSQL (Database)**:
   - The source of truth for the system.
   - Stores user information, job history, statuses, and audit timestamps.
   - Uses **golang-migrate** for versioned, reversible schema migrations.
   - Connection managed via **pgxpool** for efficient pooling under load.

4. **Redis (Queue)**:
   - Functions as a lightweight message broker.
   - Uses a **multi-queue pattern** (`queue:high`, `queue:default`, `queue:low`) for job prioritization, polled in order via `BRPOP`.
   - A separate `queue:processing` list acts as an in-flight safety net (via `BRPOPLPUSH`).
   - A `queue:dlq` list holds jobs that have exhausted all retries.
   - A **sliding window counter** per user enforces rate limiting on job submission.

---

## How It Works (Workflow)

1. **Job Submission**: A client sends a POST request to the API with a job payload. The API validates the payload, enforces rate limits, and checks for a duplicate idempotency key.
2. **Indexing**: The API saves the job in PostgreSQL and receives a unique Job ID. The row includes `created_at`, `started_at`, `finished_at`, `retry_count`, `error_message`, and an `idempotency_key`.
3. **Queuing**: The API pushes the Job ID, priority, and payload into the appropriate Redis priority queue.
4. **Acknowledgment**: The API returns an `202 Accepted` status to the client immediately, without waiting for the job to finish.
5. **Processing**: The Worker atomically moves the job from the queue into a processing list (`BRPOPLPUSH`), checks the idempotency key, updates status to `running`, and executes within a timeout context.
6. **Retry / Failure**: On failure, the worker retries with exponential backoff + jitter. After `MAX_RETRIES`, the job is moved to the DLQ and marked `failed`.
7. **Monitoring**: Clients can query the API using the Job ID to check the current status (`pending`, `running`, `completed`, `failed`, `dead`).

---

## Database Schema (Required Additions)

The `jobs` table must include the following columns for correctness and observability:

```sql
ALTER TABLE jobs
  ADD COLUMN started_at       TIMESTAMPTZ,
  ADD COLUMN finished_at      TIMESTAMPTZ,
  ADD COLUMN retry_count      INT         NOT NULL DEFAULT 0,
  ADD COLUMN max_retries      INT         NOT NULL DEFAULT 3,
  ADD COLUMN error_message    TEXT,
  ADD COLUMN idempotency_key  TEXT        UNIQUE,
  ADD COLUMN deleted_at       TIMESTAMPTZ;  -- soft deletes

-- Required indexes (without these, status/time queries are full table scans)
CREATE INDEX idx_jobs_status     ON jobs (status);
CREATE INDEX idx_jobs_created_at ON jobs (created_at);
CREATE INDEX idx_jobs_idem_key   ON jobs (idempotency_key);
```

All schema changes must be managed through **golang-migrate** migration files (never ad-hoc `ALTER TABLE` in application code).

---

## Retry & Dead Letter Queue Logic

Every job execution in the Worker must follow this flow:

```
execute(job, ctx with timeout)
  └── success  → mark completed, remove from processing queue
  └── failure  → retry_count++
                  if retry_count < max_retries:
                    re-queue with backoff delay = min(2^retry_count * base + jitter, cap)
                  else:
                    move to DLQ (queue:dlq)
                    mark status = "dead"
                    record error_message
```

The DLQ must be inspectable and replayable via an admin API endpoint (`POST /admin/jobs/:id/replay`).

---

## Worker Concurrency & Graceful Shutdown

```go
// Pseudocode — Worker startup
sem := make(chan struct{}, cfg.WorkerConcurrency) // e.g., 10

for {
  select {
  case <-shutdownCh:
    wg.Wait() // drain in-flight jobs
    return
  default:
    sem <- struct{}{}
    go func() {
      defer func() { <-sem }()
      job := popFromQueue(ctx)      // BRPOPLPUSH
      processWithTimeout(job)
    }()
  }
}
```

The service must listen for `SIGTERM` and `SIGINT` and trigger `shutdownCh` — giving in-flight jobs up to `SHUTDOWN_GRACE_PERIOD` (e.g., 30s) to finish before force-exiting.

---

## Security Requirements

- **JWT**: Implement refresh token rotation alongside short-lived access tokens.
- **Rate Limiting**: Enforce per-user job submission limits using a Redis sliding window counter (e.g., 100 jobs/minute per user).
- **Payload Validation**: Validate payload schema and reject requests exceeding `MAX_PAYLOAD_BYTES`.
- **Secrets Management**: All credentials (DB DSN, Redis URL, JWT secret) must be sourced from a secrets manager (HashiCorp Vault, AWS Secrets Manager, or Kubernetes Secrets) — never from plain `.env` files in production.
- **Admin Endpoints**: All `/admin/*` routes must require an elevated role claim in the JWT.

---

## Observability

### Structured Logging
Replace all `fmt.Print` / standard logger calls with **Zerolog** or **Zap**. Every log entry must carry:
- `job_id`, `status`, `worker_id`, `retry_count`, `duration_ms`

### Metrics (Prometheus)
Expose a `GET /metrics` endpoint and instrument the following:

| Metric | Type | Description |
|---|---|---|
| `jobs_enqueued_total` | Counter | Jobs submitted, by priority |
| `jobs_completed_total` | Counter | Jobs successfully completed |
| `jobs_failed_total` | Counter | Jobs that failed (retried) |
| `jobs_dead_total` | Counter | Jobs moved to DLQ |
| `queue_depth` | Gauge | Current pending jobs per queue |
| `job_duration_seconds` | Histogram | Execution time distribution |
| `worker_goroutines` | Gauge | Active worker goroutines |

### Health Checks
Both services must expose:
- `GET /healthz` — liveness (process is alive)
- `GET /readyz` — readiness (DB and Redis connections are healthy)

These are required for Kubernetes `livenessProbe` and `readinessProbe`.

### Distributed Tracing
Instrument both services with **OpenTelemetry**. Each trace must span from API request receipt → Redis enqueue → Worker pickup → completion/failure. Export to Jaeger or Grafana Tempo.

---

## Project Structure

```
/cmd
  /api          — API service entry point (Producer)
  /worker       — Worker service entry point (Consumer)

/internal
  /auth         — JWT authentication + refresh token logic
  /db           — pgxpool connection, query helpers
  /job          — Job creation, retrieval, retry, DLQ handlers
  /queue        — Redis queue implementation (priority queues, BRPOPLPUSH, DLQ)
  /server       — HTTP server, routing, middleware (rate limiting, auth)
  /user         — User management and registration
  /worker       — Worker pool, graceful shutdown, job executor
  /metrics      — Prometheus instrumentation
  /tracing      — OpenTelemetry setup

/migrations     — golang-migrate versioned SQL files (up + down)
/scripts        — Utility scripts (e.g., replay DLQ jobs)

docker-compose.yml   — Orchestrates API, Worker, Postgres, Redis, Prometheus, Grafana
```

---

## Getting Started

The project is containerized using Docker. To spin up the entire stack:

```bash
docker-compose up --build
```

This starts:
- **API** on `localhost:8080`
- **Worker** (background, no exposed port)
- **PostgreSQL** on `localhost:5432`
- **Redis** on `localhost:6379`
- **Prometheus** on `localhost:9090`
- **Grafana** on `localhost:3000`

Database migrations run automatically on API startup via **golang-migrate**.

---

## Key Environment Variables

| Variable | Service | Description |
|---|---|---|
| `DATABASE_URL` | API, Worker | PostgreSQL DSN (from secrets manager) |
| `REDIS_URL` | API, Worker | Redis connection string |
| `JWT_SECRET` | API | Signing secret for access tokens |
| `WORKER_CONCURRENCY` | Worker | Max concurrent jobs (default: `10`) |
| `MAX_RETRIES` | Worker | Max retry attempts before DLQ (default: `3`) |
| `MAX_PAYLOAD_BYTES` | API | Max accepted job payload size (default: `1048576`) |
| `SHUTDOWN_GRACE_PERIOD` | Worker | Seconds to wait for in-flight jobs on shutdown (default: `30`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | API, Worker | OpenTelemetry collector endpoint |

---

## Testing Requirements

- **Unit Tests**: All internal packages must have `_test.go` files covering business logic.
- **Integration Tests**: Use `testcontainers-go` to spin up real PostgreSQL and Redis instances and test the full job lifecycle (submit → queue → execute → complete/fail → DLQ).
- **Load Tests**: Run **k6** or **Vegeta** against the job submission endpoint to establish throughput baselines before production.
- **CI Pipeline**: GitHub Actions must run `go vet`, `golangci-lint`, and the full test suite on every pull request.

---

## Implementation Priority

| Priority | Change |
|---|---|
| 🔴 Critical | Retry logic + Dead Letter Queue |
| 🔴 Critical | Graceful shutdown (`SIGTERM` handling) |
| 🔴 Critical | Idempotency key check before execution |
| 🟠 High | Worker concurrency pool |
| 🟠 High | Structured logging + Prometheus metrics |
| 🟠 High | DB schema additions (audit columns + indexes) |
| 🟡 Medium | Job priority queues |
| 🟡 Medium | golang-migrate for schema management |
| 🟡 Medium | Rate limiting on submission endpoint |
| 🟢 Nice to have | OpenTelemetry distributed tracing |
| 🟢 Nice to have | Admin DLQ replay endpoint |
| 🟢 Nice to have | Scheduled / cron job support |
