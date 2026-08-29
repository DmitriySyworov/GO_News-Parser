package swaggerdocs

import (
	"errors"
	"net/http"
	"shared/loggers"
	"shared/response"
)

type HandlerSwaggerDocs struct {
	Service *ServiceSwaggerDocs
	*response.HandlerResponse
	Logger *loggers.Logger
}

func NewHandlerSwaggerDocs(router *http.ServeMux, service *ServiceSwaggerDocs, respHandler *response.HandlerResponse, logger *loggers.Logger) {
	docs := HandlerSwaggerDocs{
		Service:         service,
		HandlerResponse: respHandler,
		Logger:          logger,
	}
	router.HandleFunc("GET /api/v1/docs", docs.GetDocs())
	router.HandleFunc("GET /api/v1/docs/api", docs.GetDocsAPI())
	router.HandleFunc("PUT /api/v1/docs", docs.UpdateDocs())
}

func (h *HandlerSwaggerDocs) GetDocs() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{Error: make(map[string]string)}
		infoService := h.Service.GetDocs()
		resp.Success = true
		resp.Data = infoService
		h.ResponseSend(writer, resp, http.StatusOK)
	}
}

func (h *HandlerSwaggerDocs) GetDocsAPI() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{Error: make(map[string]string)}
		service := request.URL.Query().Get("service")
		dataDocs, errGetDocs := h.Service.GetDocsAPI(service)
		if errGetDocs != nil {
			resp.Error["docs"] = errGetDocs.Error()
			if errors.Is(errGetDocs, ErrNotFoundDocs) {
				h.ResponseSend(writer, resp, http.StatusNotFound)
			} else {
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		if _, errWrite := writer.Write(dataDocs); errWrite != nil {
			h.Logger.Error("failed to write response: " + errWrite.Error())
		}
	}
}

func (h *HandlerSwaggerDocs) UpdateDocs() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		h.Service.UpdateDocs()
		writer.WriteHeader(http.StatusNoContent)
	}
}
