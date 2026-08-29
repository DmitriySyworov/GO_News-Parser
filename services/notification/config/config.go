package notconfig

import (
	"os"
	"shared/loggers"

	"github.com/joho/godotenv"
)

type Config struct {
	*API
	*SMTP
	*Kafka
	*SharedRedis
}
type API struct {
	ApiPort string
}
type SMTP struct {
	ApiEmail        string
	ApiPassword     string
	SmtpAddress     string
	SmtpAddressHost string
}
type Kafka struct {
	Broker              string
	KafkaUser           string
	KafkaPassword       string
	NotificationTopic   string
	NotificationGroupID string
}
type SharedRedis struct {
	SharedRedisAddress  string
	SharedRedisPassword string
}

func NewConfig(logger *loggers.Logger) *Config {
	if godotenv.Load() != nil {
		logger.Warn(".env file not found. This is normal if running inside a container")
		if godotenv.Load(".env.test") != nil {
			logger.Warn(".env.test file not found. This is normal if tests don't run")
		}
	}
	apiPort := os.Getenv("EXTERNAL_API_PORT")
	apiEmail := os.Getenv("API_EMAIL")
	apiPassword := os.Getenv("API_PASSWORD")
	smtpAddress := os.Getenv("SMTP_ADDRESS")
	smtpAddressHost := os.Getenv("SMTP_ADDRESS_HOST")
	sharedRedisAddress := os.Getenv("SHARED_REDIS_ADDRESS")
	sharedRedisPassword := os.Getenv("SHARED_REDIS_PASSWORD")
	broker := os.Getenv("KAFKA_BOOTSTRAP")
	kafkaUser := os.Getenv("KAFKA_CLIENT_USER")
	kafkaPassword := os.Getenv("KAFKA_CLIENT_PASSWORD")
	notificationTopic := os.Getenv("NOTIFICATION_TOPIC")
	notificationGroupID := os.Getenv("NOTIFICATION_GROUP_ID")
	var counterEmptyVariables int
	if apiPort == "" {
		apiPort = "8080"
		logger.Warn("environment variable 'EXTERNAL_API_PORT' not found. Default value = 8080")
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
	if notificationTopic == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'NOTIFICATION_TOPIC' not found")
	}
	if notificationGroupID == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'NOTIFICATION_GROUP_ID' not found")
	}
	if apiEmail == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'API_EMAIL' not found")
	}
	if apiPassword == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'API_PASSWORD' not found")
	}
	if smtpAddress == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'SMTP_ADDRESS' not found")
	}
	if smtpAddressHost == "" {
		counterEmptyVariables++
		logger.Error("environment variable 'SMTP_ADDRESS_HOST' not found")
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
		API: &API{
			ApiPort: apiPort,
		},
		SMTP: &SMTP{
			ApiEmail:        apiEmail,
			ApiPassword:     apiPassword,
			SmtpAddress:     smtpAddress,
			SmtpAddressHost: smtpAddressHost,
		},
		Kafka: &Kafka{
			Broker:              broker,
			KafkaUser:           kafkaUser,
			KafkaPassword:       kafkaPassword,
			NotificationTopic:   notificationTopic,
			NotificationGroupID: notificationGroupID,
		},
		SharedRedis: &SharedRedis{
			SharedRedisAddress:  sharedRedisAddress,
			SharedRedisPassword: sharedRedisPassword,
		},
	}
}
