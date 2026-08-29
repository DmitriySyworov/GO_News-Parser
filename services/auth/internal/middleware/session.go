package middleware

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/appjwt"
	"context"
	"net/http"
	"shared/response"
	"shared/sherrors"
	"shared/shmiddleware"

	"github.com/google/uuid"
)

func (m *ManagerMiddleware) HandlerSessionToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			m.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + "middleware HandlerSessionToken")
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			m.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		header := request.Header.Get("X-Session-Token")
		token, errToken := shmiddleware.HelperHandleHeader(header)
		if errToken != nil {
			values.DataLog.Errors = apperrors.ErrInvalidSessionToken.Error()
			resp.Error["auth"] = apperrors.ErrInvalidSessionToken.Error()
			m.HandlerResponse.ResponseSend(writer, resp, http.StatusUnauthorized)
			return
		}
		j := appjwt.NewJWT(m.Signature, m.Logger)
		sessionID, errParse := j.ParseSessionToken(token)
		if errParse != nil {
			values.DataLog.Errors = apperrors.ErrInvalidSessionToken.Error()
			resp.Error["auth"] = apperrors.ErrInvalidSessionToken.Error()
			m.HandlerResponse.ResponseSend(writer, resp, http.StatusUnauthorized)
			return
		}
		if _, errSessionID := uuid.Parse(sessionID); errSessionID != nil {
			values.DataLog.Errors = apperrors.ErrInvalidSessionToken.Error()
			resp.Error["auth"] = apperrors.ErrInvalidSessionToken.Error()
			m.HandlerResponse.ResponseSend(writer, resp, http.StatusUnauthorized)
			return
		}
		values.DataLog.MapLog["session_id"] = sessionID
		values.DataAuth.SessionID = sessionID
		newCtxValue := context.WithValue(context.Background(), shmiddleware.KeyContextValue, values)
		ctxRequest := request.WithContext(newCtxValue)
		next.ServeHTTP(writer, ctxRequest)
	})
}
