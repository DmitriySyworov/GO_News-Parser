package authconfig

import (
	"os"
	"shared/loggers"

	"github.com/joho/godotenv"
)

type Config struct {
	*API
	*Db
	*Kafka
	*SharedRedis
}

type API struct {
	ApiPort   string
	ServiceIP string
	Signature string
}
type Db struct {
	DSN           string
	RedisAddress  string
	RedisPassword string
}
type SharedRedis struct {
	SharedRedisAddress  string
	SharedRedisPassword string
}
type Kafka struct {
	Broker            string
	KafkaUser         string
	KafkaPassword     string
	DeletedUsersTopic string
	NotificationTopic string
}

func NewConfig(logger *loggers.Logger) *Config {
	if godotenv.Load() != nil {
		logger.Warn(".env file not found. This is normal if running inside a container")
		if godotenv.Load(".env.test") != nil {
			logger.Warn(".env.test file not found. This is normal if tests don't run")
		}
	}
	apiPort := os.Getenv("EXTERNAL_API_PORT")
	serviceIP := os.Getenv("SERVICE_IP")
	signature := os.Getenv("JWT_SIGNATURE")
	dsn := os.Getenv("DSN")
	redisAddress := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	broker := os.Getenv("KAFKA_BOOTSTRAP")
	kafkaUser := os.Getenv("KAFKA_CLIENT_USER")
	kafkaPassword := os.Getenv("KAFKA_CLIENT_PASSWORD")
	deletedUsersTopic := os.Getenv("DELETED_USERS_TOPIC")
	sharedRedisAddress := os.Getenv("SHARED_REDIS_ADDRESS")
	sharedRedisPassword := os.Getenv("SHARED_REDIS_PASSWORD")
	notificationTopic := os.Getenv("NOTIFICATION_TOPIC")
	var counterEmptyVariables int
	if apiPort == "" {
		apiPort = "8080"
		logger.Warn("environment variable 'EXTERNAL_API_PORT' not found. Default value = 8080")
	}
	if serviceIP == "" {
		serviceIP = "localhost"
		logger.Warn("environment variable 'SERVICE_IP' not found. Default value = localhost")
	}
	if dsn == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'DSN' not found")
	}
	if redisAddress == "" {
		redisAddress = "localhost:6379"
		logger.Error("environment variable 'REDIS_ADDRESS' not found. Default value = localhost:6379")
	}
	if redisPassword == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'REDIS_PASSWORD' not found")
	}
	if broker == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'KAFKA_BOOTSTRAP' not found")
	}
	if kafkaUser == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'KAFKA_CLIENT_USER' not found")
	}
	if kafkaPassword == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'KAFKA_CLIENT_PASSWORD' not found")
	}
	if deletedUsersTopic == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'DELETED_USERS_TOPIC' not found")
	}
	if notificationTopic == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'NOTIFICATION_TOPIC' not found")
	}
	if signature == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'JWT_SIGNATURE' not found")
	}
	if sharedRedisAddress == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'SHARED_REDIS_ADDRESS' not found")
	}
	if sharedRedisPassword == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'SHARED_REDIS_PASSWORD' not found")
	}
	if counterEmptyVariables != 0 {
		os.Exit(1)
	}
	return &Config{
		Db: &Db{
			DSN:           dsn,
			RedisPassword: redisPassword,
			RedisAddress:  redisAddress,
		},
		API: &API{
			ServiceIP: serviceIP,
			ApiPort:   apiPort,
			Signature: signature,
		},
		Kafka: &Kafka{
			Broker:            broker,
			KafkaUser:         kafkaUser,
			KafkaPassword:     kafkaPassword,
			DeletedUsersTopic: deletedUsersTopic,
			NotificationTopic: notificationTopic,
		},
		SharedRedis: &SharedRedis{
			SharedRedisAddress:  sharedRedisAddress,
			SharedRedisPassword: sharedRedisPassword,
		},
	}
}
