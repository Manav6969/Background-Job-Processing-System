package worker

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/logger"
	"github.com/Manav6969/Background-Job-Processing-System/internal/metrics"
	"github.com/Manav6969/Background-Job-Processing-System/internal/queue"
)

const (
	processingQueue = "jobs_processing"
	dlqQueue        = "jobs_dlq"
)

type Job struct {
	ID             int         `json:"id"`
	Type           string      `json:"type"`
	Payload        interface{} `json:"payload"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
	Priority       string      `json:"priority,omitempty"`
}

type Pool struct {
	concurrency int
	queue       *queue.RedisQueue
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewPool(concurrency int, q *queue.RedisQueue) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		concurrency: concurrency,
		queue:       q,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (p *Pool) Start() {
	sem := make(chan struct{}, p.concurrency)

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			default:
				sem <- struct{}{}

				msg, err := p.queue.PriorityPop(p.ctx, processingQueue, "high", "default", "low")
				if err != nil {
					<-sem
					continue
				}

				p.wg.Add(1)
				metrics.WorkerGoroutines.Inc()
				
				go func(message string) {
					defer p.wg.Done()
					defer metrics.WorkerGoroutines.Dec()
					defer func() { <-sem }()

					p.processJob(message)
				}(msg)
			}
		}
	}()
}

func (p *Pool) Stop(gracePeriod int) {
	log := logger.Log
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All workers finished. Exiting.")
	case <-time.After(time.Duration(gracePeriod) * time.Second):
		log.Warn().Int("timeout_seconds", gracePeriod).Msg("Shutdown timeout reached. Forcing exit.")
	}
}

func (p *Pool) processJob(msg string) {
	log := logger.Log
	start := time.Now()

	var job Job
	if err := json.Unmarshal([]byte(msg), &job); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal job")
		_ = p.queue.Acknowledge(p.ctx, processingQueue, msg)
		return
	}

	jobLog := log.With().
		Int("job_id", job.ID).
		Str("job_type", job.Type).
		Logger()

	// Idempotency check
	var currentStatus string
	err := db.Pool.QueryRow(p.ctx, "SELECT status FROM jobs WHERE id=$1", job.ID).Scan(&currentStatus)
	if err != nil {
		jobLog.Error().Err(err).Msg("Failed to check job status")
		_ = p.queue.Acknowledge(p.ctx, processingQueue, msg)
		return
	}
	if currentStatus == "completed" || currentStatus == "dead" {
		jobLog.Info().Str("status", currentStatus).Msg("Skipping already processed job")
		_ = p.queue.Acknowledge(p.ctx, processingQueue, msg)
		return
	}

	// Update status to running
	_, _ = db.Pool.Exec(p.ctx,
		"UPDATE jobs SET status='running', started_at=NOW() WHERE id=$1",
		job.ID,
	)

	jobLog.Info().Msg("Processing job")

	// Execute with timeout
	execCtx, execCancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer execCancel()

	err = p.executeJob(execCtx, job)
	duration := time.Since(start)
	metrics.JobDurationSeconds.Observe(duration.Seconds())

	if err != nil {
		jobLog.Warn().
			Err(err).
			Dur("duration_ms", duration).
			Msg("Job failed")
		p.handleFailure(msg, job, err)
		return
	}

	// Success
	_, _ = db.Pool.Exec(p.ctx,
		"UPDATE jobs SET status='completed', finished_at=NOW() WHERE id=$1",
		job.ID,
	)
	_ = p.queue.Acknowledge(p.ctx, processingQueue, msg)
	metrics.JobsCompletedTotal.Inc()
	jobLog.Info().Dur("duration_ms", duration).Msg("Job completed")
}

func (p *Pool) executeJob(ctx context.Context, job Job) error {
	select {
	case <-time.After(2 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) handleFailure(msg string, job Job, jobErr error) {
	log := logger.Log

	var retryCount, maxRetries int
	err := db.Pool.QueryRow(p.ctx,
		"SELECT retry_count, max_retries FROM jobs WHERE id=$1",
		job.ID,
	).Scan(&retryCount, &maxRetries)
	if err != nil {
		log.Error().Err(err).Int("job_id", job.ID).Msg("Failed to get retry info")
		return
	}

	retryCount++

	if retryCount >= maxRetries {
		log.Error().
			Int("job_id", job.ID).
			Int("retry_count", retryCount).
			Int("max_retries", maxRetries).
			Msg("Max retries exhausted, moving to DLQ")

		_, _ = db.Pool.Exec(p.ctx,
			"UPDATE jobs SET status='dead', retry_count=$1, error_message=$2, finished_at=NOW() WHERE id=$3",
			retryCount, jobErr.Error(), job.ID,
		)
		_ = p.queue.MoveToDLQ(p.ctx, processingQueue, dlqQueue, msg)
		metrics.JobsDeadTotal.Inc()
		return
	}

	backoff := calculateBackoff(retryCount)
	log.Info().
		Int("job_id", job.ID).
		Int("attempt", retryCount).
		Int("max_retries", maxRetries).
		Dur("backoff", backoff).
		Msg("Retrying job")

	_, _ = db.Pool.Exec(p.ctx,
		"UPDATE jobs SET status='failed', retry_count=$1, error_message=$2 WHERE id=$3",
		retryCount, jobErr.Error(), job.ID,
	)

	_ = p.queue.Acknowledge(p.ctx, processingQueue, msg)
	metrics.JobsFailedTotal.Inc()
	time.Sleep(backoff)

	_, _ = db.Pool.Exec(p.ctx,
		"UPDATE jobs SET status='pending' WHERE id=$1",
		job.ID,
	)
	
	priority := job.Priority
	if priority == "" {
		priority = "default"
	}
	_ = p.queue.PushWithPriority(p.ctx, msg, priority)
}

func calculateBackoff(retryCount int) time.Duration {
	base := 1.0
	maxBackoff := 60.0
	exp := math.Min(math.Pow(2, float64(retryCount))*base, maxBackoff)
	jitter := rand.Float64() * exp * 0.5
	return time.Duration((exp + jitter) * float64(time.Second))
}
