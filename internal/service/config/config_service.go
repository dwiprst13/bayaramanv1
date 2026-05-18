package config

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type ConfigService interface {
	GetPlatformFeePercent(ctx context.Context) float64
	SetPlatformFeePercent(ctx context.Context, percent float64) error
	GetWithdrawDelayHours(ctx context.Context) int
	SetWithdrawDelayHours(ctx context.Context, hours int) error
	GetEscrowExpiryHours(ctx context.Context) int
	SetEscrowExpiryHours(ctx context.Context, hours int) error
	GetAllConfigs(ctx context.Context) map[string]interface{}
}

type configService struct {
	redisClient *redis.Client
}

func NewConfigService(redisClient *redis.Client) ConfigService {
	return &configService{redisClient: redisClient}
}

func (s *configService) GetPlatformFeePercent(ctx context.Context) float64 {
	val, err := s.redisClient.Get(ctx, "config:platform_fee_percent").Result()
	if err != nil {
		return 5.0 // Default 5%
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 5.0
	}
	return parsed
}

func (s *configService) SetPlatformFeePercent(ctx context.Context, percent float64) error {
	return s.redisClient.Set(ctx, "config:platform_fee_percent", percent, 0).Err()
}

func (s *configService) GetWithdrawDelayHours(ctx context.Context) int {
	val, err := s.redisClient.Get(ctx, "config:withdraw_delay_hours").Result()
	if err != nil {
		return 24 // Default 24 hours
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 24
	}
	return parsed
}

func (s *configService) SetWithdrawDelayHours(ctx context.Context, hours int) error {
	return s.redisClient.Set(ctx, "config:withdraw_delay_hours", hours, 0).Err()
}

func (s *configService) GetEscrowExpiryHours(ctx context.Context) int {
	val, err := s.redisClient.Get(ctx, "config:escrow_expiry_hours").Result()
	if err != nil {
		return 24 // Default 24 hours
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 24
	}
	return parsed
}

func (s *configService) SetEscrowExpiryHours(ctx context.Context, hours int) error {
	return s.redisClient.Set(ctx, "config:escrow_expiry_hours", hours, 0).Err()
}

func (s *configService) GetAllConfigs(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"platform_fee_percent": s.GetPlatformFeePercent(ctx),
		"withdraw_delay_hours": s.GetWithdrawDelayHours(ctx),
		"escrow_expiry_hours":  s.GetEscrowExpiryHours(ctx),
	}
}
