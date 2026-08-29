package ratelimit

import (
	"context"
	"shared/loggers"
	"shared/shconstant"
	"shared/storage"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	Redis  *storage.Redis
	Logger *loggers.Logger
}

const (
	KeyRateLimiting       = "ratelimit:user:"
	RequestLimit    int64 = 100
)

func NewRateLimiter(rdb *storage.Redis, logger *loggers.Logger) *Limiter {
	return &Limiter{
		Redis:  rdb,
		Logger: logger,
	}
}
func (l *Limiter) RateLimiter(ctxRequest context.Context, key string) (int64, map[string]string, error) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	clearBefore := now - int64(time.Minute/time.Millisecond)
	keyRateLimit := KeyRateLimiting + key
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	pipe := l.Redis.Pipeline()
	pipe.ZRemRangeByScore(ctxTimeout, keyRateLimit, "0", strconv.FormatInt(clearBefore, 10))
	pipe.ZAdd(ctxTimeout, keyRateLimit, redis.Z{
		Score:  float64(now),
		Member: uuid.New().String(),
	})
	zCardCmd := pipe.ZCard(ctxTimeout, keyRateLimit)
	pipe.Expire(ctxTimeout, keyRateLimit, 2*time.Minute)
	_, errPipeline := pipe.Exec(ctxTimeout)
	if errPipeline != nil {
		l.Logger.Error("failed to rate limiting: " + errPipeline.Error())
		return 0, nil, errPipeline
	}
	counterRequest := zCardCmd.Val()
	var remaining string
	if counterRequest >= RequestLimit {
		remaining = "0"
	} else {
		remaining = strconv.FormatInt(RequestLimit-counterRequest, 10)
	}
	headers := map[string]string{
		"X-RateLimit-Limit":     strconv.FormatInt(RequestLimit, 10),
		"X-RateLimit-Remaining": remaining,
		"X-RateLimit-Reset":     strconv.FormatInt(time.Now().Add(1*time.Minute).Unix(), 10),
	}
	return counterRequest, headers, nil
}
