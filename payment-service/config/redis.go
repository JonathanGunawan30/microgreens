package config

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		TLSConfig:    &tls.Config{},
		DialTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	return rdb
}
