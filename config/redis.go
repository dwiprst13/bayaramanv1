package config

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *Config) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Ping Redis
	err = client.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	log.Println("Redis connection established")
	return client, nil
}
