package config

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(addr string, db int, ctx context.Context) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	return client
}
