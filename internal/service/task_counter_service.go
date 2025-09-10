package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// TaskCounterService manages Redis-based counters for email task progress tracking.
type TaskCounterService interface {
	InitializeTaskCounter(int64, int) error
	DecrementTaskCounter(int64) (int, error)
	GetTaskCounter(int64) (int, error)
	DeleteTaskCounter(int64) error
}

type taskCounterService struct {
	redisClient *redis.Client
}

// NewTaskCounterService creates a new TaskCounterService instance.
func NewTaskCounterService(redisClient *redis.Client) TaskCounterService {
	return &taskCounterService{
		redisClient: redisClient,
	}
}

// getCounterKey returns the Redis key for a task counter.
func (s *taskCounterService) getCounterKey(taskID int64) string {
	return fmt.Sprintf("task:counter:%d", taskID)
}

// InitializeTaskCounter sets the initial counter value for a task.
func (s *taskCounterService) InitializeTaskCounter(taskID int64, totalCount int) error {
	ctx := context.Background()
	key := fmt.Sprintf("task:counter:%d", taskID)

	// Set the counter with a TTL of 24 hours to prevent memory leaks
	err := s.redisClient.Set(ctx, key, totalCount, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to initialize task counter for task %d: %w", taskID, err)
	}

	return nil
}

// DecrementTaskCounter decrements the counter by 1 and returns the remaining count.
func (s *taskCounterService) DecrementTaskCounter(taskID int64) (int, error) {
	ctx := context.Background()
	key := s.getCounterKey(taskID)

	// Use DECR to atomically decrement the counter
	result, err := s.redisClient.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement task counter for task %d: %w", taskID, err)
	}

	// Ensure counter doesn't go below 0
	if result < 0 {
		// Reset to 0 if it went negative (shouldn't happen in normal cases)
		s.redisClient.Set(ctx, key, 0, 24*time.Hour)
		return 0, nil
	}

	return int(result), nil
}

// GetTaskCounter returns the current counter value for a task.
func (s *taskCounterService) GetTaskCounter(taskID int64) (int, error) {
	ctx := context.Background()
	key := s.getCounterKey(taskID)

	val, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Counter doesn't exist
		return 0, fmt.Errorf("task counter for task %d not found", taskID)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get task counter for task %d: %w", taskID, err)
	}

	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid counter value for task %d: %w", taskID, err)
	}

	return count, nil
}

// DeleteTaskCounter removes the counter for a task.
func (s *taskCounterService) DeleteTaskCounter(taskID int64) error {
	ctx := context.Background()
	key := s.getCounterKey(taskID)

	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete task counter for task %d: %w", taskID, err)
	}

	return nil
}
