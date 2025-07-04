package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	"email-service/internal/config"
)

// NewClient creates and returns a new Redis client.
// It uses the application's configuration to connect to Redis.
func NewClient(cfg *config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping the Redis server to ensure the connection is working.
	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return rdb, nil
}
