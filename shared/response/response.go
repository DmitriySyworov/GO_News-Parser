package response

import (
	"encoding/json"
	"shared/loggers"

	"net/http"
)

type Response struct {
	Success bool
	Data    any               `json:"data,omitempty"`
	Error   map[string]string `json:"errors,omitempty"`
}
type NegativeResponse struct {
	Success bool              `json:"success" example:"false"`
	Error   map[string]string `json:"errors,omitempty" swaggertype:"object"`
}

type HandlerResponse struct {
	*loggers.Logger
}

func NewHandlerResponse(logger *loggers.Logger) *HandlerResponse {
	return &HandlerResponse{
		Logger: logger,
	}
}

func (hr *HandlerResponse) ResponseSend(writer http.ResponseWriter, response *Response, status int) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	errEncode := json.NewEncoder(writer).Encode(response)
	if errEncode != nil {
		hr.Logger.Error("failed to process the response: " + errEncode.Error())
	}
}
