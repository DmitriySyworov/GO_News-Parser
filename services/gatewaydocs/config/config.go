package docsconfig

import (
	"os"
	"shared/loggers"

	"github.com/joho/godotenv"
)

type Config struct {
	*Api
	*Services
}
type Api struct {
	ApiPort string
}
type Services struct {
	AuthUserURL string
}

func NewConfig(logger *loggers.Logger) *Config {
	if godotenv.Load() != nil {
		logger.Warn(".env file not found.  This is normal if running inside a container")
		if godotenv.Load(".env.test") != nil {
			logger.Warn(".env.test file not found. This is normal if tests don't run")
		}
	}
	apiPort := os.Getenv("EXTERNAL_API_PORT")
	authUserURL := os.Getenv("AUTH_USER_URL")
	counterEmptyVariables := 0
	if apiPort == "" {
		apiPort = "8080"
		logger.Warn("environment variable 'EXTERNAL_API_PORT' not found. Default value = 8080")
	}
	if authUserURL == "" {
		logger.Error("environment variable 'AUTH_USER_URL' not found")
		counterEmptyVariables++
	}
	if counterEmptyVariables != 0 {
		os.Exit(1)
	}
	return &Config{
		Api: &Api{
			ApiPort: apiPort,
		},
		Services: &Services{
			AuthUserURL: authUserURL,
		},
	}
}
