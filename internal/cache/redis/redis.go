package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new redis cache that complies with cache interface
func NewRedisCache(ctx context.Context, addr string) (*RedisCache, error) {
	rClient := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	retryTicker := time.NewTicker(time.Second * 2)
	defer retryTicker.Stop()

	// retry ping
	var pingErr error
	for range 5 {
		if pingErr = rClient.Ping(ctx).Err(); pingErr == nil {
			break
		}
		<-retryTicker.C
	}
	if pingErr != nil {
		return nil, fmt.Errorf("failed to ping redis instance: %w", pingErr)
	}

	return &RedisCache{
		client: rClient,
	}, nil
}

func (r *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}
