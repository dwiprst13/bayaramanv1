package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiterService interface {
	CheckLimit(ctx context.Context, action string, identifier string, maxRequests int, window time.Duration) error
	ResetLimit(ctx context.Context, action string, identifier string) error
}

type rateLimiterService struct {
	redisClient *redis.Client
}

func NewRateLimiterService(redisClient *redis.Client) RateLimiterService {
	return &rateLimiterService{redisClient: redisClient}
}

func (s *rateLimiterService) CheckLimit(ctx context.Context, action string, identifier string, maxRequests int, window time.Duration) error {
	key := fmt.Sprintf("ratelimit:%s:%s", action, identifier)
	
	count, err := s.redisClient.Get(ctx, key).Int()
	if err == nil && count >= maxRequests {
		return errors.New("rate limit exceeded, please wait")
	}

	if err == redis.Nil || count == 0 {
		s.redisClient.Set(ctx, key, 1, window).Err()
	} else {
		s.redisClient.Incr(ctx, key).Err()
	}

	return nil
}

func (s *rateLimiterService) ResetLimit(ctx context.Context, action string, identifier string) error {
	key := fmt.Sprintf("ratelimit:%s:%s", action, identifier)
	return s.redisClient.Del(ctx, key).Err()
}
