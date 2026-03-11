package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements sliding window rate limiting using Redis.
type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks if a request is allowed under the rate limit.
// key is typically "action:identifier" (e.g., "login:192.168.1.1").
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()
	rkey := fmt.Sprintf("ratelimit:%s", key)

	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, rkey, "-inf", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, rkey)
	pipe.ZAdd(ctx, rkey, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, rkey, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := countCmd.Val()
	return count < int64(limit), nil
}
