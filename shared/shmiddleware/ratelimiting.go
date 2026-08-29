package shmiddleware

import (
	"net/http"
	"shared/ratelimit"
	"shared/response"
	"shared/sherrors"
	"strings"
)

func (m *ManagerSharedMiddleware) RateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/ready") || strings.Contains(request.URL.Path, "/health") {
			next.ServeHTTP(writer, request)
			return
		}
		resp := &response.Response{Error: make(map[string]string, 1)}
		ctxValues := request.Context().Value(KeyContextValue)
		values, ok := ctxValues.(*ContextValues)
		if !ok {
			m.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			m.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		counterRequest, headers, errRateLimiting := m.RateLimit.RateLimiter(request.Context(), values.DataAuth.UserUUID)
		if errRateLimiting != nil {
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			m.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		if counterRequest > ratelimit.RequestLimit {
			writer.Header().Set("Retry-After", "60")
			for key, value := range headers {
				writer.Header().Set(key, value)
			}
			resp.Error["rate_limit"] = sherrors.ErrRateLimiting.Error()
			m.ResponseSend(writer, resp, http.StatusTooManyRequests)
			return
		}
		rateLimitWrapperWriter := &RateLimitWrapperWriter{
			ResponseWriter: writer,
			Headers:        headers,
		}
		next.ServeHTTP(rateLimitWrapperWriter, request)
	})
}

type RateLimitWrapperWriter struct {
	http.ResponseWriter
	Headers map[string]string
}

func (w *RateLimitWrapperWriter) WriteHeader(code int) {
	for key, value := range w.Headers {
		w.ResponseWriter.Header().Set(key, value)
	}
	w.ResponseWriter.WriteHeader(code)
}
