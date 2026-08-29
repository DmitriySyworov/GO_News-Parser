package integration

import (
	"context"
	"errors"
	"os"
	"shared/loggers"
	"shared/testutil"
	"testing"
	"time"
)

var (
	suiteContainer = struct {
		*testutil.TestContainerRedis
		*testutil.TestContainerPostgres
	}{}
	logger     *loggers.Logger
	errInitial error
)

func TestMain(t *testing.M) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	testContainerPostgres, errInitPostgres := testutil.NewTestContainerPostgres(ctxTimeout, "../../cmd/migration/sql")
	if errInitPostgres != nil {
		errInitial = errInitPostgres
	}
	testContainerRedis, errInitRedis := testutil.NewTestContainerRedis(ctxTimeout)
	if errInitRedis != nil {
		errInitial = errors.Join(errInitial, errInitRedis)
	}
	logger = loggers.NewLogger()
	suiteContainer.TestContainerRedis = testContainerRedis
	suiteContainer.TestContainerPostgres = testContainerPostgres
	code := t.Run()
	os.Exit(code)
}
