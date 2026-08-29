package shmiddleware

import (
	"net/http"
	"shared/response"
	"shared/sherrors"
)

func (m *ManagerSharedMiddleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if errPanic := recover(); errPanic != nil {
				resp := &response.Response{
					Error: make(map[string]string),
				}
				resp.Error["global"] = sherrors.ErrCriticalServer.Error()
				m.ResponseSend(writer, resp, http.StatusInternalServerError)
				m.Logger.Error("critical error: ", "panic", errPanic)
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
