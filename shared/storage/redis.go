package storage

import (
	"shared/loggers"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	*redis.Client
}

func OpenRedis(redisAddress, redisPassword string, loggers *loggers.Logger) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr:            redisAddress,
		Password:        redisPassword,
		DB:              0,
		PoolSize:        100,
		MinIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	})
	loggers.Info("connect redis successful")
	return &Redis{
		Client: rdb,
	}
}
