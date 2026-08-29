package testutil

import (
	"context"
	"errors"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/lib/pq"
)

type TestContainerPostgres struct {
	Postgres *gorm.DB
}
type TestContainerRedis struct {
	Redis *redis.Client
}

func NewTestContainerPostgres(ctx context.Context, pathMigrationFile string) (*TestContainerPostgres, error) {
	postgresReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "admin_test",
				"POSTGRES_PASSWORD": "password_test",
				"POSTGRES_DB":       "container_test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	}
	testPostgres, errContainerPostgres := testcontainers.GenericContainer(ctx, postgresReq)
	if errContainerPostgres != nil {
		return nil, errors.New("failed to start container PostgreSQL: " + errContainerPostgres.Error())
	}
	postgresPort, errGetPostgresPort := testPostgres.MappedPort(ctx, "5432")
	if errGetPostgresPort != nil {
		return nil, errors.New("failed to get port PostgresSQL: " + errGetPostgresPort.Error())
	}
	dsn := "host=127.0.0.1 user=admin_test password=password_test dbname=container_test port=" + postgresPort.Port() + " sslmode=disable"
	gormDB, errOpenGorm := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if errOpenGorm != nil {
		return nil, errors.New("failed to open gorm: " + errOpenGorm.Error())
	}
	sqlDriver, errExtractSqlDriver := gormDB.DB()
	if errExtractSqlDriver != nil {
		return nil, errors.New("failed to extract sql driver: " + errExtractSqlDriver.Error())
	}
	if errSetDialect := goose.SetDialect("postgres"); errSetDialect != nil {
		return nil, errors.New("failed to set dialect PostgreSQL: " + errSetDialect.Error())
	}
	if errMigrate := goose.Up(sqlDriver, pathMigrationFile); errMigrate != nil {
		return nil, errors.New("failed to migrate tables: " + errMigrate.Error())
	}
	return &TestContainerPostgres{
		Postgres: gormDB,
	}, nil
}
func NewTestContainerRedis(ctx context.Context) (*TestContainerRedis, error) {
	redisReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	}
	testRedis, errContainerRedis := testcontainers.GenericContainer(ctx, redisReq)
	if errContainerRedis != nil {
		return nil, errors.New("failed to start container Redis: " + errContainerRedis.Error())
	}
	redisPort, errGetRedisPort := testRedis.MappedPort(ctx, "6379")
	if errGetRedisPort != nil {
		return nil, errors.New("failed to get redis port: " + errGetRedisPort.Error())
	}
	redisAddr := "127.0.0.1:" + redisPort.Port()
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &TestContainerRedis{
		Redis: rdb,
	}, nil
}
