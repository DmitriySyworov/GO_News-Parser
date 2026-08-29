package shmiddleware

import (
	"shared/loggers"
	"shared/ratelimit"
	"shared/response"
)

type ManagerSharedMiddleware struct {
	Logger    *loggers.Logger
	Signature string
	*response.HandlerResponse
	RateLimit *ratelimit.Limiter
}

type ContextValues struct {
	DataAuth *DataAuth
	DataLog  *DataLog
}

const (
	KeyContextValue = "keyCtxValue"
)

func NewManagerSharedMiddleware(signature string, rateLimiter *ratelimit.Limiter, logger *loggers.Logger, handlerResponse *response.HandlerResponse) *ManagerSharedMiddleware {
	return &ManagerSharedMiddleware{
		Signature:       signature,
		Logger:          logger,
		HandlerResponse: handlerResponse,
		RateLimit:       rateLimiter,
	}
}
