package shmiddleware

import (
	"context"
	"net/http"
	"shared/loggers"
	"strings"
)

type DataLog struct {
	Method   string
	Path     string
	UserUUID string
	MapLog   map[string]any
	Errors   string
}

func (m *ManagerSharedMiddleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/health") ||
			strings.Contains(request.URL.Path, "/ready") {
			next.ServeHTTP(writer, request)
			return
		}
		wrapperWriter := &WrapperWriter{
			Status:         http.StatusOK,
			ResponseWriter: writer,
		}
		const sizeMap = 8
		logValues := &ContextValues{
			DataAuth: &DataAuth{},
			DataLog: &DataLog{
				MapLog: make(map[string]any, sizeMap),
			}}
		ctxValue := context.WithValue(context.Background(), KeyContextValue, logValues)
		ctxRequest := request.WithContext(ctxValue)
		logValues.DataLog.Method = request.Method
		logValues.DataLog.Path = request.URL.Path
		next.ServeHTTP(wrapperWriter, ctxRequest)
		dataLoggerHandler := &loggers.DataLoggerHandler{
			Status:      wrapperWriter.Status,
			Method:      logValues.DataLog.Method,
			Path:        logValues.DataLog.Path,
			UserUUID:    logValues.DataLog.UserUUID,
			Errors:      logValues.DataLog.Errors,
			DataRequest: logValues.DataLog.MapLog,
		}
		if wrapperWriter.Status >= 200 && wrapperWriter.Status < 400 {
			dataLoggerHandler.Msg = "successful operation"
			m.Logger.LoggerHandler(dataLoggerHandler)
		} else if wrapperWriter.Status >= 400 && wrapperWriter.Status < 500 {
			dataLoggerHandler.Msg = "unsuccessful operation"
			m.Logger.LoggerHandler(dataLoggerHandler)
		} else {
			dataLoggerHandler.Msg = "critical error"
			m.Logger.LoggerHandler(dataLoggerHandler)
		}
	})
}

type WrapperWriter struct {
	http.ResponseWriter
	Status int
}

func (w *WrapperWriter) WriteHeader(code int) {
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}
