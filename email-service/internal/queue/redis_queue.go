package queue

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisQueueService implements the QueueService interface using Redis.
type RedisQueueService struct {
	client *redis.Client
}

// NewRedisQueueService creates a new Redis-backed QueueService.
func NewRedisQueueService(client *redis.Client) QueueService {
	return &RedisQueueService{
		client: client,
	}
}

// Enqueue adds a message to a Redis list using LPUSH.
func (s *RedisQueueService) Enqueue(ctx context.Context, queueName string, message string) error {
	return s.client.LPush(ctx, queueName, message).Err()
}

// Dequeue removes and returns a message from a Redis list using BRPOP.
// It blocks until a message is available or the context is cancelled.
// The timeout of 0 means it will block indefinitely.
func (s *RedisQueueService) Dequeue(ctx context.Context, queueName string) (string, error) {
	result, err := s.client.BRPop(ctx, 0*time.Second, queueName).Result()
	if err != nil {
		return "", err
	}
	// BRPop returns a slice with the queue name and the value.
	// e.g., ["myqueue", "message-body"]
	return result[1], nil
}
