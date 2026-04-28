package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	name   string
}

func NewRedisQueue(addr string, name string) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisQueue{
		client: rdb,
		name:   name,
	}
}

func (q *RedisQueue) Push(ctx context.Context, job string) error {
	return q.client.LPush(ctx, q.name+":default", job).Err()
}

func (q *RedisQueue) PushWithPriority(ctx context.Context, job string, priority string) error {
	queueName := q.name + ":default"
	if priority != "" {
		queueName = q.name + ":" + priority
	}
	return q.client.LPush(ctx, queueName, job).Err()
}

func (q *RedisQueue) PriorityPop(ctx context.Context, processingQueue string, priorities ...string) (string, error) {
	queues := make([]string, len(priorities))
	for i, p := range priorities {
		queues[i] = q.name + ":" + p
	}
	
	// Use BRPop to block on multiple priority queues
	result, err := q.client.BRPop(ctx, 1*time.Second, queues...).Result()
	if err != nil {
		return "", err
	}
	
	job := result[1]
	
	// Push to processing queue to maintain in-flight tracking
	_ = q.client.LPush(ctx, processingQueue, job).Err()
	
	return job, nil
}

func (q *RedisQueue) ReliablePop(ctx context.Context, processingQueue string) (string, error) {
	// Fallback to old behavior for backward compatibility or default queue
	result, err := q.client.BRPopLPush(ctx, q.name+":default", processingQueue, 1*time.Second).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (q *RedisQueue) Acknowledge(ctx context.Context, processingQueue string, job string) error {
	return q.client.LRem(ctx, processingQueue, 1, job).Err()
}

func (q *RedisQueue) MoveToDLQ(ctx context.Context, processingQueue string, dlqName string, job string) error {
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, processingQueue, 1, job)
	pipe.LPush(ctx, dlqName, job)
	_, err := pipe.Exec(ctx)
	return err
}

func (q *RedisQueue) GetClient() *redis.Client {
	return q.client
}
