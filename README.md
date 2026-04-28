<p align="center">
  <h1 align="center">⚙️ Background Job Processing System</h1>
  <p align="center">
    A distributed, production-grade background job processing system built with <strong>Go</strong>, <strong>PostgreSQL</strong>, and <strong>Redis</strong>.
    <br />
    Implements a robust producer-consumer architecture with priority queues, exponential backoff retries, a Dead Letter Queue, and full Prometheus observability.
  </p>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/PostgreSQL-14-336791?style=for-the-badge&logo=postgresql&logoColor=white" />
  <img src="https://img.shields.io/badge/Redis-6-DC382D?style=for-the-badge&logo=redis&logoColor=white" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white" />
  <img src="https://img.shields.io/badge/Prometheus-Metrics-E6522C?style=for-the-badge&logo=prometheus&logoColor=white" />
</p>

---

## 📋 Table of Contents

- [Architecture](#-architecture)
- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Project Structure](#-project-structure)
- [Getting Started](#-getting-started)
- [API Reference](#-api-reference)
- [Job Lifecycle](#-job-lifecycle)
- [Priority Queues](#-priority-queues)
- [Retry & Dead Letter Queue](#-retry--dead-letter-queue)
- [Observability](#-observability)
- [Environment Variables](#-environment-variables)
- [Testing](#-testing)
- [Contributing](#-contributing)

---

## 🏗 Architecture

```
┌───────────────┐          ┌─────────────────┐          ┌──────────────────┐
│               │  HTTP    │                 │  Redis   │                  │
│    Client     │────────▶ │   API Service   │────────▶ │  Worker Service  │
│   (cURL/UI)   │  REST    │   (Producer)    │  Queue   │   (Consumer)     │
│               │ ◀──────  │                 │          │                  │
└───────────────┘  202     └────────┬────────┘          └────────┬─────────┘
                                   │                             │
                                   │  SQL                        │  SQL
                                   ▼                             ▼
                           ┌─────────────────┐
                           │                 │
                           │   PostgreSQL    │
                           │  (Source of     │
                           │   Truth)        │
                           │                 │
                           └─────────────────┘
```

**Flow:**
1. Client submits a job via REST API → API validates, persists to PostgreSQL, and pushes to Redis.
2. Client receives an immediate `202 Accepted` response.
3. Worker atomically pops from Redis priority queues and processes the job.
4. On success → marks `completed`. On failure → retries with exponential backoff or moves to DLQ.

---

## ✨ Features

| Category | Feature | Status |
|---|---|---|
| **Core** | Producer-Consumer architecture | ✅ |
| **Core** | Priority queues (high / default / low) | ✅ |
| **Core** | Idempotency key (duplicate prevention) | ✅ |
| **Reliability** | Exponential backoff with jitter | ✅ |
| **Reliability** | Dead Letter Queue (DLQ) | ✅ |
| **Reliability** | Reliable queue pop (`BRPOPLPUSH`) | ✅ |
| **Reliability** | Graceful shutdown (`SIGTERM`/`SIGINT`) | ✅ |
| **Reliability** | Configurable worker concurrency pool | ✅ |
| **Reliability** | Job execution timeout (30s) | ✅ |
| **Security** | JWT authentication | ✅ |
| **Security** | Redis sliding-window rate limiting | ✅ |
| **Security** | Payload size validation (1MB limit) | ✅ |
| **Security** | Bcrypt password hashing | ✅ |
| **Observability** | Structured logging (Zerolog) | ✅ |
| **Observability** | Prometheus metrics (`/metrics`) | ✅ |
| **Observability** | Health checks (`/healthz`, `/readyz`) | ✅ |
| **Operations** | Docker Compose orchestration | ✅ |
| **Operations** | Versioned DB migrations (golang-migrate) | ✅ |
| **Admin** | DLQ replay endpoint | ✅ |

---

## 🛠 Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.25 |
| Web Framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL 14 via [pgxpool](https://github.com/jackc/pgx) |
| Queue / Cache | Redis 6 via [go-redis](https://github.com/redis/go-redis) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Auth | [golang-jwt](https://github.com/golang-jwt/jwt) + bcrypt |
| Logging | [Zerolog](https://github.com/rs/zerolog) |
| Metrics | [Prometheus client_golang](https://github.com/prometheus/client_golang) |
| Containers | Docker & Docker Compose |

---

## 📁 Project Structure

```
.
├── cmd/
│   ├── api/
│   │   └── main.go              # API service entry point (Producer)
│   └── worker/
│       └── main.go              # Worker service entry point (Consumer)
│
├── internal/
│   ├── auth/
│   │   └── jwt.go               # JWT token generation
│   ├── config/
│   │   └── config.go            # Environment-based configuration
│   ├── db/
│   │   └── postgres.go          # pgxpool connection + golang-migrate runner
│   ├── job/
│   │   └── handler.go           # Job CRUD handlers + DLQ replay
│   ├── logger/
│   │   └── logger.go            # Zerolog structured logger setup
│   ├── metrics/
│   │   └── metrics.go           # Prometheus counters, gauges, histograms
│   ├── queue/
│   │   └── redis.go             # Redis queue (priority push/pop, DLQ, ack)
│   ├── server/
│   │   ├── server.go            # HTTP server, routing, middleware wiring
│   │   ├── jwt.go               # JWT auth middleware
│   │   └── ratelimit.go         # Redis sliding-window rate limiter
│   ├── user/
│   │   └── handler.go           # User registration & login
│   └── worker/
│       └── worker.go            # Worker pool, job processor, retry logic
│
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_jobs_table.up.sql
│   └── 000002_create_jobs_table.down.sql
│
├── scripts/
│   └── test_workflow.sh         # End-to-end integration test script
│
├── docker-compose.yml           # Orchestrates API, Worker, Postgres, Redis
├── Dockerfile.api               # API container build
├── Dockerfile.worker            # Worker container build
├── go.mod
├── go.sum
└── understanding.md             # Architectural design document
```

---

## 🚀 Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- (Optional) [Go 1.25+](https://go.dev/dl/) for local development

### Run with Docker Compose

```bash
# Clone the repository
git clone https://github.com/Manav6969/Background-Job-Processing-System.git
cd Background-Job-Processing-System

# Start all services
docker-compose up --build -d

# Verify everything is running
docker-compose ps
```

This starts:
| Service | Port | Description |
|---|---|---|
| **API** | `localhost:6969` | REST API (Producer) |
| **Worker** | — (no port) | Background processor (Consumer) |
| **PostgreSQL** | `localhost:5432` | Database |
| **Redis** | `localhost:6379` | Message queue |

### Verify Health

```bash
# Liveness
curl http://localhost:6969/healthz
# → {"status":"alive"}

# Readiness (checks DB + Redis)
curl http://localhost:6969/readyz
# → {"db":"up","redis":"up","status":"ready"}
```

### Tear Down

```bash
docker-compose down -v   # -v removes volumes (clears DB data)
```

---

## 📡 API Reference

### Authentication

#### Register a User
```bash
curl -X POST http://localhost:6969/register \
  -H "Content-Type: application/json" \
  -d '{"username": "manav", "password": "secret123"}'
```
```json
{"id": 1, "username": "manav", "message": "user registered successfully"}
```

#### Login (Get JWT)
```bash
curl -X POST http://localhost:6969/login \
  -H "Content-Type: application/json" \
  -d '{"username": "manav", "password": "secret123"}'
```
```json
{"token": "eyJhbGciOi...", "user_id": 1, "username": "manav"}
```

> Use the returned `token` as a `Bearer` token in all authenticated requests.

---

### Jobs

#### Submit a Job
```bash
curl -X POST http://localhost:6969/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -d '{
    "type": "email_send",
    "payload": {"to": "user@example.com", "subject": "Hello!"},
    "priority": "high",
    "idempotency_key": "email-001"
  }'
```
```json
{"id": 1, "status": "queued"}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | string | ✅ | Job type identifier (e.g., `email_send`, `report`) |
| `payload` | object | ✅ | Arbitrary JSON payload (max 1MB) |
| `priority` | string | ❌ | `high`, `default` (default), or `low` |
| `idempotency_key` | string | ❌ | Unique key to prevent duplicate processing |

#### Get Job Status
```bash
curl http://localhost:6969/jobs/1 \
  -H "Authorization: Bearer <YOUR_TOKEN>"
```
```json
{"id": "1", "status": "completed", "type": "email_send", "retry_count": 0}
```

Possible statuses: `pending` → `running` → `completed` | `failed` → `dead`

---

### Admin

#### Replay a Dead Job
Re-enqueues a job from the Dead Letter Queue back into the `high` priority queue.

```bash
curl -X POST http://localhost:6969/admin/jobs/5/replay \
  -H "Authorization: Bearer <YOUR_TOKEN>"
```
```json
{"message": "job replayed successfully"}
```

> Only jobs with status `dead` can be replayed. All others return a `400` error.

---

### Health & Observability

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/healthz` | GET | ❌ | Liveness probe (is the process alive?) |
| `/readyz` | GET | ❌ | Readiness probe (are DB and Redis connected?) |
| `/metrics` | GET | ❌ | Prometheus metrics endpoint |

---

## 🔄 Job Lifecycle

```
                    ┌──────────┐
     Submit ──────▶ │ pending  │
                    └────┬─────┘
                         │  Worker picks up
                         ▼
                    ┌──────────┐
                    │ running  │
                    └────┬─────┘
                    ╱          ╲
              Success          Failure
                ╱                  ╲
     ┌────────────┐          ┌──────────┐
     │ completed  │          │  failed  │ ──── retry_count < max_retries
     └────────────┘          └────┬─────┘      → re-queue with backoff
                                  │
                            retry_count >= max_retries
                                  │
                                  ▼
                             ┌─────────┐
                             │  dead   │ ──── moved to DLQ
                             └─────────┘      (replayable via admin API)
```

---

## 🎯 Priority Queues

Jobs are enqueued into separate Redis lists based on their priority level:

| Priority | Redis Key | Behavior |
|---|---|---|
| 🔴 High | `jobs:high` | Processed first |
| 🟡 Default | `jobs:default` | Processed after high queue is empty |
| 🟢 Low | `jobs:low` | Processed last |

The worker uses `BRPOP` across all three lists in priority order, ensuring that high-priority jobs always preempt lower-priority ones.

---

## 🔁 Retry & Dead Letter Queue

When a job fails:

1. `retry_count` is incremented.
2. If `retry_count < max_retries` (default: 3):
   - Job is re-queued with **exponential backoff + jitter**.
   - Backoff formula: `min(2^retry_count × base + jitter, 60s)`
3. If `retry_count >= max_retries`:
   - Job status is set to `dead`.
   - Job is moved to the **Dead Letter Queue** (`jobs_dlq` in Redis).
   - Error message is recorded in the database.

Dead jobs can be inspected and replayed via `POST /admin/jobs/:id/replay`.

---

## 📊 Observability

### Prometheus Metrics

Available at `GET /metrics`:

| Metric | Type | Description |
|---|---|---|
| `jobs_enqueued_total` | Counter | Total jobs submitted, labeled by priority |
| `jobs_completed_total` | Counter | Jobs successfully completed |
| `jobs_failed_total` | Counter | Jobs that failed (will be retried) |
| `jobs_dead_total` | Counter | Jobs moved to the Dead Letter Queue |
| `job_duration_seconds` | Histogram | Execution time distribution |
| `worker_goroutines` | Gauge | Current active worker goroutines |

### Structured Logging

All logs use [Zerolog](https://github.com/rs/zerolog) with structured JSON fields:

```
2026-04-28T19:30:00Z INF Processing job  job_id=1 job_type=email_send caller=worker.go:120
2026-04-28T19:30:02Z INF Job completed   job_id=1 duration_ms=2001.5 caller=worker.go:145
```

### Health Probes

Designed for Kubernetes `livenessProbe` and `readinessProbe`:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 6969
readinessProbe:
  httpGet:
    path: /readyz
    port: 6969
```

---

## ⚙️ Environment Variables

| Variable | Service | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | API, Worker | `postgres://postgres:postgres@localhost:5432/...` | PostgreSQL connection string |
| `REDIS_ADDR` | API, Worker | `localhost:6379` | Redis host:port |
| `JWT_SECRET` | API | `default_secret_change_me` | HMAC signing secret for JWTs |
| `PORT` | API | `:6969` | HTTP listen address |
| `RATE_LIMIT` | API | `5` | Max requests per minute per user/IP |
| `MIGRATIONS_PATH` | API | `./migrations` | Path to SQL migration files |
| `WORKER_CONCURRENCY` | Worker | `10` | Max concurrent goroutines |
| `SHUTDOWN_GRACE_PERIOD` | Worker | `30` | Seconds to drain in-flight jobs on shutdown |
| `LOG_LEVEL` | Both | `info` | Set to `debug` for verbose logging |

---

## 🧪 Testing

### Automated Integration Test

A full end-to-end test script is included:

```bash
# Make sure the stack is running first
docker-compose up --build -d

# Run the test suite
bash scripts/test_workflow.sh
```

This script tests:
- ✅ Health check endpoints
- ✅ User registration & login
- ✅ Job submission (high, default, low priority)
- ✅ Idempotency (duplicate detection)
- ✅ Worker processing & job completion
- ✅ DLQ replay guard (rejects non-dead jobs)
- ✅ Prometheus metrics exposure

### Manual Testing with cURL

```bash
# 1. Register
curl -X POST http://localhost:6969/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'

# 2. Login
TOKEN=$(curl -s -X POST http://localhost:6969/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"//')

# 3. Submit a job
curl -X POST http://localhost:6969/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type":"test","payload":{"hello":"world"},"priority":"high"}'

# 4. Check status
curl http://localhost:6969/jobs/1 -H "Authorization: Bearer $TOKEN"

# 5. View metrics
curl http://localhost:6969/metrics | grep jobs_
```

---

## 🗄 Database Schema

### Users Table
```sql
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Jobs Table
```sql
CREATE TABLE jobs (
    id              SERIAL PRIMARY KEY,
    user_id         INT REFERENCES users(id),
    type            TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          job_status NOT NULL DEFAULT 'pending',  -- enum: pending, running, completed, failed, dead
    idempotency_key TEXT UNIQUE,
    retry_count     INT NOT NULL DEFAULT 0,
    max_retries     INT NOT NULL DEFAULT 3,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ
);

-- Performance indexes
CREATE INDEX idx_jobs_status     ON jobs (status);
CREATE INDEX idx_jobs_created_at ON jobs (created_at);
CREATE INDEX idx_jobs_idem_key   ON jobs (idempotency_key);
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

---

<p align="center">
  Built with ❤️ by <a href="https://github.com/Manav6969">Manav</a>
</p>
