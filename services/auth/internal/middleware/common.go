package middleware

import (
	"shared/loggers"
	"shared/response"
)

type ManagerMiddleware struct {
	Logger    *loggers.Logger
	Signature string
	*response.HandlerResponse
}

func NewManagerMiddleware(signature string, logger *loggers.Logger, handlerResponse *response.HandlerResponse) *ManagerMiddleware {
	return &ManagerMiddleware{
		Signature:       signature,
		Logger:          logger,
		HandlerResponse: handlerResponse,
	}
}
