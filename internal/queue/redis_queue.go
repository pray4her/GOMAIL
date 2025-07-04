package queue

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// var (
// 	TaskCreatedQueue   = "tasks:created"
// 	TaskScheduledQueue = "tasks:scheduled"
// 	EmailSendingQueue  = "email:sending"
// 	RecipientSyncQueue = "recipients:sync" // Queue for syncing recipients to ES
// )

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

// EnqueueScheduled adds a message to a Redis sorted set (zset) with its score.
func (s *RedisQueueService) EnqueueScheduled(ctx context.Context, queueName string, message string, score float64) error {
	return s.client.ZAdd(ctx, queueName, &redis.Z{
		Score:  score,
		Member: message,
	}).Err()
}

// DequeueDue retrieves and removes all messages from a sorted set whose score is up to the given max score.
// It uses a Lua script to ensure atomicity of the ZRangeByScore and ZRemRangeByScore operations.
func (s *RedisQueueService) DequeueDue(ctx context.Context, queueName string, maxScore float64) ([]string, error) {
	// Lua script to atomically get and remove due items.
	// KEYS[1] is the sorted set name.
	// ARGV[1] is the max score (e.g., current timestamp).
	script := `
local items = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
if #items > 0 then
    redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
end
return items
`
	keys := []string{queueName}
	argv := []interface{}{maxScore}

	result, err := s.client.Eval(ctx, script, keys, argv...).Result()
	if err != nil {
		// If the script returns nothing (no items due), Redis might return a Nil error, which is not a "real" error.
		if err == redis.Nil {
			return []string{}, nil
		}
		return nil, err
	}

	// The Lua script returns an array of items, which needs to be type-asserted.
	items, ok := result.([]interface{})
	if !ok {
		return []string{}, nil // Return empty slice if result is not what we expect
	}

	dueItems := make([]string, len(items))
	for i, item := range items {
		dueItems[i], _ = item.(string)
	}

	return dueItems, nil
}
