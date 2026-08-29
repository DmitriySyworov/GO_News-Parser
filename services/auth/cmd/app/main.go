package main

import (
	authconfig "app/auth-service/config"
	"app/auth-service/internal/auth"
	"app/auth-service/internal/di"
	"app/auth-service/internal/listener"
	"app/auth-service/internal/middleware"
	"app/auth-service/internal/notifer"
	"app/auth-service/internal/user"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shared/loggers"
	"shared/outbox"
	"shared/ratelimit"
	"shared/response"
	"shared/shkafka"
	"shared/shmiddleware"
	"shared/storage"
	"syscall"
	"time"

	"app/auth-service/docs"

	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title           Auth & User Microservice API
// @version         1.0
// @description     Core authentication and identity service with token rotation, profile management, and sliding window rate limiting.
// @host            localhost:8080
// @BasePath        /api/v1
// @tag.name        auth
// @tag.description User registration, authentication, and session management
// @tag.name        user
// @tag.description Profile management, account data retrieval, and user deletion logs
func main() {
	appVariable := App()
	server := http.Server{
		Addr:    ":" + appVariable.Conf.ApiPort,
		Handler: appVariable.HandlerApp,
	}

	ctxCancel, cancelCancel := context.WithCancel(context.Background())
	go Run(ctxCancel, appVariable.Notifer)
	serverError := make(chan error, 1)
	stopSignal := make(chan os.Signal, 1)
	go func() {
		if errServer := server.ListenAndServe(); errServer != nil && !errors.Is(errServer, http.ErrServerClosed) {
			serverError <- errServer
		}
	}()
	signal.Notify(stopSignal, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-stopSignal:
		appVariable.Logger.Info("received stop signal, starting graceful shutdown")
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if errShutdown := server.Shutdown(ctxShutdown); errShutdown != nil {
			appVariable.Logger.Error("failed graceful shutdown: " + errShutdown.Error())
		}
		cancel()
	case err := <-serverError:
		appVariable.Logger.Error("error on the server: " + err.Error())
	}
	cancelCancel()
	appVariable.KafkaProducerDelete.CloseProducer()
	appVariable.KafkaProducerEmail.CloseProducer()
	if errCloseRedis := appVariable.Redis.Close(); errCloseRedis != nil {
		appVariable.Logger.Error("failed to close redis: " + errCloseRedis.Error())
	}
	if errCloseSharedRedis := appVariable.SharedRedis.Close(); errCloseSharedRedis != nil {
		appVariable.Logger.Error("failed to close shared redis: " + errCloseSharedRedis.Error())
	}
	sqlDb, errGetDriverDb := appVariable.Postgres.DB.DB()
	if errGetDriverDb != nil {
		appVariable.Logger.Error("failed to extract sql driver: " + errGetDriverDb.Error())
		return
	}
	if errClosePostgres := sqlDb.Close(); errClosePostgres != nil {
		appVariable.Logger.Error("failed to close Postgres: " + errClosePostgres.Error())
	}
}

type AppVariable struct {
	Conf                *authconfig.Config
	KafkaProducerDelete *shkafka.KafkaProducer
	KafkaProducerEmail  *shkafka.KafkaProducer
	SharedRedis         *storage.Redis
	Redis               *storage.Redis
	Postgres            *storage.Postgres
	Logger              *loggers.Logger
	HandlerApp          http.Handler
	ServiceUser         *user.ServiceUser
	IRepoUser           di.IRepoUser
	Notifer             *notifer.Notifer
	ExpenseListener     *listener.KafkaListener
}

func App() *AppVariable {
	logging := loggers.NewLogger()
	//
	conf := authconfig.NewConfig(logging)
	//
	validate := validator.New()
	//
	responseHandler := response.NewHandlerResponse(logging)
	//
	sharedRedis := storage.OpenRedis(conf.SharedRedisAddress, conf.SharedRedisPassword, logging)
	rateLimiter := ratelimit.NewRateLimiter(sharedRedis, logging)
	//
	sharedMv := shmiddleware.NewManagerSharedMiddleware(conf.Signature, rateLimiter, logging, responseHandler)
	mv := middleware.NewManagerMiddleware(conf.Signature, logging, responseHandler)
	//
	postgres := storage.OpenPostgres(conf.DSN, logging)
	redis := storage.OpenRedis(conf.RedisAddress, conf.RedisPassword, logging)
	producerDeleteEvent, errInitialProducerDelete := shkafka.NewProducer(&shkafka.ConfigProducer{
		Brokers:       []string{conf.Broker},
		KafkaUser:     conf.KafkaUser,
		KafkaPassword: conf.KafkaPassword,
		Topic:         conf.DeletedUsersTopic,
	}, logging)
	if errInitialProducerDelete != nil {
		logging.Error("failed to initial producer delete event kafka: " + errInitialProducerDelete.Error())
		os.Exit(1)
	}
	producerEmailEvent, errInitProducerEmail := shkafka.NewProducer(&shkafka.ConfigProducer{
		Brokers:       []string{conf.Broker},
		KafkaUser:     conf.KafkaUser,
		KafkaPassword: conf.KafkaPassword,
		Topic:         conf.NotificationTopic,
	}, logging)
	if errInitProducerEmail != nil {
		logging.Error("failed to init producer email event kafka: " + errInitProducerEmail.Error())
		os.Exit(1)
	}
	//
	router := http.NewServeMux()
	//
	repoAuth := auth.NewRepositoryRedis(redis, logging)
	repoUser := user.NewRepositoryUser(postgres, logging)
	//
	box := outbox.NewOutbox(sharedRedis, logging)
	nt := notifer.NewNotifer(producerEmailEvent, box, logging)
	listenerExpenseEvent := listener.NewKafkaListener(producerEmailEvent, sharedRedis, repoUser, logging)
	//
	serviceAuth := auth.NewServiceAuth(repoAuth, nt, repoUser, conf, logging)
	serviceUser := user.NewServiceUser(repoUser, serviceAuth, repoAuth, producerDeleteEvent, conf.Signature, logging)
	//
	docs.SwaggerInfo.Host = conf.ServiceIP + ":" + conf.ApiPort
	router.Handle("GET /swagger/{any...}", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	router.HandleFunc("GET /health", health(logging))
	router.HandleFunc("GET /ready", ready(postgres, redis, sharedRedis, logging))
	auth.NewHandlerAuth(router, serviceAuth, responseHandler, validate, rateLimiter, logging, mv)
	user.NewHandlerUser(router, serviceUser, responseHandler, validate, logging, mv, sharedMv)
	//
	chainMv := shmiddleware.Chain(
		sharedMv.Logging,
		sharedMv.Recovery,
	)
	return &AppVariable{
		Conf:                conf,
		KafkaProducerDelete: producerDeleteEvent,
		KafkaProducerEmail:  producerEmailEvent,
		Logger:              logging,
		HandlerApp:          chainMv(router),
		Postgres:            postgres,
		Redis:               redis,
		SharedRedis:         sharedRedis,
		ServiceUser:         serviceUser,
		IRepoUser:           repoUser,
		Notifer:             nt,
		ExpenseListener:     listenerExpenseEvent,
	}
}
func Run(ctxCancel context.Context, nt *notifer.Notifer) {
	go nt.SenderSecurityLater(ctxCancel)
}

func health(logger *loggers.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		if _, errWrite := writer.Write([]byte("OK")); errWrite != nil {
			logger.Error("failed to writer health check: " + errWrite.Error())
		}
	}
}
func ready(postgres *storage.Postgres, redis, sharedRedis *storage.Redis, logger *loggers.Logger) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		sqlDb, errDb := postgres.DB.DB()
		if errDb != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			logger.Error("ready check failed (Postgres init): " + errDb.Error())
			return
		}
		if errPingPostgres := sqlDb.PingContext(ctxTimeout); errPingPostgres != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			logger.Error("ready check failed (Postgres ping): " + errPingPostgres.Error())
			return
		}
		if errPingRedis := redis.Ping(ctxTimeout).Err(); errPingRedis != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			logger.Error("ready check failed (Redis ping): " + errPingRedis.Error())
			return
		}
		if errPingSharedRedis := sharedRedis.Ping(ctxTimeout).Err(); errPingSharedRedis != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			logger.Error("ready check failed (shared Redis ping): " + errPingSharedRedis.Error())
			return
		}
		writer.WriteHeader(http.StatusOK)
		if _, errWrite := writer.Write([]byte("READY")); errWrite != nil {
			logger.Error("failed to write ready check: " + errWrite.Error())
			return
		}
	}
}
