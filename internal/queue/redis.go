package queue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

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

func (q *RedisQueue) Push(job string) error {
	return q.client.LPush(ctx, q.name, job).Err()
}

func (q *RedisQueue) Pop() (string, error) {
	result, err := q.client.BRPop(ctx, 0, q.name).Result()
	if err != nil {
		return "", err
	}
	
	return result[1], nil
}
