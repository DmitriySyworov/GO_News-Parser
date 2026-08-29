package testutil

import (
	"context"
	"shared/shconstant"
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func CleanRedis(t *testing.T, rdb *redis.Client) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutRedis)
	defer cancel()
	if errFlushAll := rdb.FlushAll(ctxTimeout).Err(); errFlushAll != nil {
		t.Fatal("failed to clean redis: ", errFlushAll)
	}
}
func CleanPostgres(t *testing.T, postgres *gorm.DB, tables []string) {
	var queryTruncate string
	for _, table := range tables {
		queryTruncate += "TRUNCATE TABLE " + table + "; "
	}
	if errCleanPostgres := postgres.Exec(queryTruncate).Error; errCleanPostgres != nil {
		t.Fatal("failed to clean postgres: ", errCleanPostgres)
	}
}
