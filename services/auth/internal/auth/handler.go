package auth

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/common"
	"app/auth-service/internal/ip"
	"app/auth-service/internal/middleware"
	"context"
	"errors"
	"net/http"
	"shared/loggers"
	"shared/ratelimit"
	"shared/requestutil"
	"shared/response"
	"shared/sherrors"
	"shared/shmiddleware"

	_ "app/auth-service/internal/common"

	"github.com/go-playground/validator/v10"
)

type HandlerAuth struct {
	*response.HandlerResponse
	*loggers.Logger
	Validate *validator.Validate
	*ratelimit.Limiter
	*ServiceAuth
}

func NewHandlerAuth(router *http.ServeMux, service *ServiceAuth, handlerResponse *response.HandlerResponse, validate *validator.Validate, rateLimiter *ratelimit.Limiter, logger *loggers.Logger, mv *middleware.ManagerMiddleware) {
	auth := &HandlerAuth{
		ServiceAuth:     service,
		Validate:        validate,
		HandlerResponse: handlerResponse,
		Limiter:         rateLimiter,
		Logger:          logger,
	}
	router.HandleFunc("POST /api/v1/register", auth.Register())
	router.HandleFunc("POST /api/v1/login", auth.Login())
	router.HandleFunc("POST /api/v1/recovery", auth.Recovery())
	router.Handle("POST /api/v1/confirm", mv.HandlerSessionToken(auth.Confirm()))
	router.HandleFunc("POST /api/v1/refresh", auth.Refresh())
	router.HandleFunc("POST /api/v1/logout", auth.Logout())
}

// Register godoc
// @Summary      Register a new user
// @Description  Validates name length, strong password policy, and email uniqueness before triggering internal registration workflow.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RequestRegister  true  "Registration payload"
// @Success      202      {object}  response.Response{data=common.ResponseAuth,errors=nil} "User registration process accepted"
// @Failure      400      {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"name\": \"name must be between 2 and 64 characters\" } } or { \"errors\": { \"email\": \"incorrect email\" } } or { \"errors\": { \"password\": \"password must be between 8 and 24 characters\" } } or { \"errors\": { \"user\": \"user already exist\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to ensure security\" } }"
// @Router       /register [post]
func (h *HandlerAuth) Register() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestRegister](request.Body, h.Validate)
		if errBody != nil {
			mapError := sherrors.MapError{Map: make(map[string]string, 3)}
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					switch {
					case err.Field() == "Name":
						values.DataLog.MapLog["name"] = body.Name
						mapError.Map["name"] = ErrIncorrectName.Error()
					case err.Field() == "Email":
						values.DataLog.MapLog["email"] = body.Email
						mapError.Map["email"] = apperrors.ErrIncorrectEmail.Error()
					case err.Field() == "Password":
						mapError.Map["password"] = apperrors.ErrIncorrectEnterPassword.Error()
					}
				}
			} else {
				mapError.Map["body"] = errBody.Error()
			}
			values.DataLog.Errors = mapError.Error()
			resp.Error = mapError.Map
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		if !h.helperRateLimiting(request.Context(), body.Email, writer) {
			return
		}
		respAuth, errAuth := h.ServiceAuth.Register(request.Context(), body)
		if errAuth != nil {
			values.DataLog.Errors = errAuth.Error()
			var mapError sherrors.MapError
			if errors.As(errAuth, &mapError) {
				resp.Error = mapError.Map
				if mapError.Map["global"] == "" {
					h.ResponseSend(writer, resp, http.StatusBadRequest)
				} else {
					h.ResponseSend(writer, resp, http.StatusInternalServerError)
				}
				return
			}
			if errors.Is(errAuth, apperrors.ErrFailedSecurity) {
				resp.Error["global"] = errAuth.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respAuth
		h.ResponseSend(writer, resp, http.StatusAccepted)
	}
}

// Login godoc
// @Summary      Authenticate user
// @Description  Verifies credentials using bcrypt against PostgreSQL and issues a secure session token pair via internal workflow.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RequestLogin  true  "Login credentials"
// @Success      202      {object}  response.Response{data=common.ResponseAuth,errors=nil} "Authentication accepted"
// @Failure      400      {object}  response.NegativeResponse "Validation errors. Format: { \"errors\": { \"email\": \"incorrect email\" } } or { \"errors\": { \"password\": \"password must be between 8 and 24 characters\" } }"
// @Failure      401      {object}  response.NegativeResponse "Incorrect credentials. Format: { \"errors\": { \"auth\": \"incorrect password or email\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to ensure security\" } }"
// @Router       /login [post]
func (h *HandlerAuth) Login() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestLogin](request.Body, h.Validate)
		if errBody != nil {
			mapError := sherrors.MapError{Map: make(map[string]string, 2)}
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					switch {
					case err.Field() == "Email":
						values.DataLog.MapLog["email"] = body.Email
						mapError.Map["email"] = apperrors.ErrIncorrectEmail.Error()
					case err.Field() == "Password":
						mapError.Map["password"] = apperrors.ErrIncorrectEnterPassword.Error()
					}
				}
			} else {
				mapError.Map["body"] = errBody.Error()
			}
			values.DataLog.Errors = mapError.Error()
			resp.Error = mapError.Map
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		if !h.helperRateLimiting(request.Context(), body.Email, writer) {
			return
		}
		respAuth, errLogin := h.ServiceAuth.Login(request.Context(), body)
		if errLogin != nil {
			values.DataLog.Errors = errLogin.Error()
			switch {
			case errors.Is(errLogin, apperrors.ErrIncorrectPasswordOrEmail):
				resp.Error["auth"] = errLogin.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			default:
				resp.Error["global"] = errLogin.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respAuth
		h.ResponseSend(writer, resp, http.StatusAccepted)
	}
}

// Recovery godoc
// @Summary      Recover user account or password
// @Description  Initiates recovery process (recovery_password or recovery_user) via email verification. For account recovery, requires active password validation.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        action   query     string  true  "Type of recovery: recovery_password or recovery_user"
// @Param        request  body      RequestRecovery  true  "Recovery payload (Password is required only for recovery_user)"
// @Success      202      {object}  response.Response{data=common.ResponseAuth,errors=nil} "Recovery process accepted"
// @Failure      400      {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"email\": \"incorrect email\" } } or { \"errors\": { \"action\": \"action must be recovery_password or recovery_user\" } }"
// @Failure      401      {object}  response.NegativeResponse "Authentication errors. Format: { \"errors\": { \"auth\": \"password is empty\" } } or { \"errors\": { \"auth\": \"incorrect password or email\" } }"
// @Failure      404      {object}  response.NegativeResponse "Data errors. Format: { \"errors\": { \"email\": \"not found user\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to ensure security\" } }"
// @Router       /recovery [post]
func (h *HandlerAuth) Recovery() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestRecovery](request.Body, h.Validate)
		if errBody != nil {
			mapError := sherrors.MapError{Map: make(map[string]string, 2)}
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					switch {
					case err.Field() == "Email":
						values.DataLog.MapLog["email"] = body.Email
						mapError.Map["email"] = apperrors.ErrIncorrectEmail.Error()
					case err.Field() == "Password":
						mapError.Map["password"] = apperrors.ErrIncorrectEnterPassword.Error()
					}
				}
			} else {
				mapError.Map["body"] = errBody.Error()
			}
			values.DataLog.Errors = mapError.Error()
			resp.Error = mapError.Map
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		if !h.helperRateLimiting(request.Context(), body.Email, writer) {
			return
		}
		action := request.URL.Query().Get("action")
		values.DataLog.MapLog["action"] = action
		respAuth, errRecovery := h.ServiceAuth.Recovery(request.Context(), body, action)
		if errRecovery != nil {
			values.DataLog.Errors = errRecovery.Error()
			var mapError sherrors.MapError
			if errors.As(errRecovery, &mapError) {
				resp.Error = mapError.Map
				switch {
				case apperrors.ErrNotFoundUser.Error() == mapError.Map["email"] && len(mapError.Map) == 1:
					h.ResponseSend(writer, resp, http.StatusNotFound)
				default:
					h.ResponseSend(writer, resp, http.StatusBadRequest)
				}
				return
			}
			switch {
			case errors.Is(errRecovery, apperrors.ErrFailedSecurity):
				resp.Error["global"] = errRecovery.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			default:
				resp.Error["auth"] = errRecovery.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			}
			return
		}
		resp.Success = true
		resp.Data = respAuth
		h.ResponseSend(writer, resp, http.StatusAccepted)
	}
}

// Confirm godoc
// @Summary      Confirm OTP code for registration, login or recovery
// @Description  Verifies the 6-digit OTP code against the active Redis session. If valid, completes the requested action (creates user, updates password, or activates profile) and issues session tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        X-Session-Token   header    string  true  "Bearer <session_token> (Validated via Session Middleware)"
// @Param        action          query     string  true  "Workflow action: register, login, recovery_user, or recovery_password"
// @Param        request         body      RequestConfirm  true  "Confirmation payload (NewPassword is required only for recovery_password)"
// @Success      201             {object}  response.Response{data=ResponseConfirm,errors=nil} "Successfully created (for action=register) or successfully processed (200 OK for other actions)"
// @Failure      400             {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"code\": \"the code must be 6 characters\" } } or { \"errors\": { \"action\": \"the action must be register, login or recovery\" } } or { \"errors\": { \"session\": \"incorrect session id\" } } or { \"errors\": { \"user\": \"user already exist\" } } or { \"errors\": { \"new_password\": \"new password not specified\" } }"
// @Failure      401             {object}  response.NegativeResponse "Authentication or session errors. Format: { \"errors\": { \"auth\": \"invalid session token\" } } or { \"errors\": { \"auth\": \"authorization session has expired or does not exist\" } } or { \"errors\": { \"auth\": \"incorrect code\" } }"
// @Failure      404             {object}  response.NegativeResponse "Data errors. Format: { \"errors\": { \"user\": \"not found user\" } }"
// @Failure      429             {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500             {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to ensure security\" } }"
// @Router       /confirm [post]
func (h *HandlerAuth) Confirm() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestConfirm](request.Body, h.Validate)
		if errBody != nil {
			mapError := sherrors.MapError{Map: make(map[string]string, 2)}
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					if err.Field() == "Code" {
						mapError.Map["code"] = apperrors.ErrIncorrectFormatCode.Error()
					}
					if err.Field() == "NewPassword" {
						mapError.Map["new_password"] = apperrors.ErrIncorrectEnterNewPassword.Error()
					}
				}
			} else {
				mapError.Map["body"] = errBody.Error()
			}
			values.DataLog.Errors = mapError.Error()
			resp.Error = mapError.Map
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		action := request.URL.Query().Get("action")
		values.DataLog.MapLog["action"] = action
		userAgent := request.Header.Get("User-Agent")
		values.DataLog.MapLog["user_agent"] = userAgent
		if !h.helperRateLimiting(request.Context(), values.DataAuth.SessionID, writer) {
			return
		}
		ipUser := ip.GetIP(request)
		values.DataLog.MapLog["ip"] = ipUser
		respConfirm, errConfirm := h.ServiceAuth.Confirm(request.Context(), body, values.DataAuth.SessionID, action, userAgent, ipUser)
		if errConfirm != nil {
			values.DataLog.Errors = errConfirm.Error()
			var mapError sherrors.MapError
			if errors.As(errConfirm, &mapError) {
				resp.Error = mapError.Map
				if mapError.Map[common.AttemptsLeft] != "" {
					h.ResponseSend(writer, resp, http.StatusUnauthorized)
				} else {
					h.ResponseSend(writer, resp, http.StatusBadRequest)
				}
				return
			}
			switch {
			case errors.Is(errConfirm, ErrUserAlreadyExist):
				resp.Error["user"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusBadRequest)
			case errors.Is(errConfirm, ErrNotSpecifiedNewPassword):
				resp.Error["new_password"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusBadRequest)
			case errors.Is(errConfirm, apperrors.ErrSessionExpired), errors.Is(errConfirm, apperrors.ErrIncorrectCode):
				resp.Error["auth"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			case errors.Is(errConfirm, apperrors.ErrNotFoundUser):
				resp.Error["user"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusNotFound)
			default:
				resp.Error["global"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respConfirm
		if action == ActionRegister {
			h.ResponseSend(writer, resp, http.StatusCreated)
		} else {
			h.ResponseSend(writer, resp, http.StatusOK)
		}
	}
}

// Refresh godoc
// @Summary      Rotate refresh token
// @Description  Parses old appjwt refresh token from body, validates session UUID and User-Agent in Redis, and performs safe Token Rotation using WATCH locking.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RequestRefreshLogout  true  "Refresh token payload"
// @Success      200      {object}  response.Response{data=ResponseConfirm,errors=nil} "Tokens successfully rotated"
// @Failure      400      {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"refresh_jwt\": \"refresh token not sent\" } } or { \"errors\": { \"body\": \"invalid json format\" } }"
// @Failure      401      {object}  response.NegativeResponse "Token or session expired errors. Format: { \"errors\": { \"refresh_token\": \"refresh token renewal error\" } } or { \"errors\": { \"refresh_token\": \"session has expired try again later\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } }"
// @Router       /refresh [post]
func (h *HandlerAuth) Refresh() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestRefreshLogout](request.Body, h.Validate)
		if errBody != nil {
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					if err.Field() == "RefreshJwt" {
						values.DataLog.Errors = apperrors.ErrSentRefresh.Error()
						resp.Error["refresh_jwt"] = apperrors.ErrSentRefresh.Error()
					}
				}
			} else {
				values.DataLog.Errors = errBody.Error()
				resp.Error["body"] = errBody.Error()
			}
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		if !h.helperRateLimiting(request.Context(), body.RefreshJwt, writer) {
			return
		}
		userAgent := request.Header.Get("User-Agent")
		values.DataLog.MapLog["user_agent"] = userAgent
		ipUser := ip.GetIP(request)
		values.DataLog.MapLog["ip"] = ipUser
		respConfirm, errRefresh := h.ServiceAuth.Refresh(request.Context(), body.RefreshJwt, userAgent, ipUser)
		if errRefresh != nil {
			values.DataLog.Errors = errRefresh.Error()
			switch {
			case errors.Is(errRefresh, apperrors.ErrRenewalRefresh), errors.Is(errRefresh, ErrRefreshExpired):
				resp.Error["refresh_token"] = errRefresh.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			default:
				resp.Error["global"] = errRefresh.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respConfirm
		h.ResponseSend(writer, resp, http.StatusOK)
	}
}

// Logout godoc
// @Summary      Logout user and invalidate session
// @Description  Parses the refresh token from request body, extracts session UUID, and safely deletes the active session from Redis.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        all      query     bool    false "If true, terminates all active sessions across all devices. Default is false (only current device)." default(false)
// @Param        request  body      RequestRefreshLogout  true  "Logout payload containing refresh token"
// @Success      204      "Session successfully invalidated, no content returned"
// @Failure      400      {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"refresh_token\": \"refresh token not sent\" } } or { \"errors\": { \"body\": \"invalid json format\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } }"
// @Router       /logout [post]
func (h *HandlerAuth) Logout() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		resp := &response.Response{
			Error: make(map[string]string),
		}
		ctxValues := request.Context().Value(shmiddleware.KeyContextValue)
		values, ok := ctxValues.(*shmiddleware.ContextValues)
		if !ok {
			h.Logger.Error(sherrors.ErrFailedAssertionContextValues.Error() + request.Pattern)
			resp.Error["global"] = sherrors.ErrCriticalServer.Error()
			h.ResponseSend(writer, resp, http.StatusInternalServerError)
			return
		}
		body, errBody := requestutil.HandlerRequest[RequestRefreshLogout](request.Body, h.Validate)
		if errBody != nil {
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					if err.Field() == "RefreshJwt" {
						values.DataLog.Errors = apperrors.ErrSentRefresh.Error()
						resp.Error["refresh_jwt"] = apperrors.ErrSentRefresh.Error()
					}
				}
			} else {
				values.DataLog.Errors = errBody.Error()
				resp.Error["body"] = errBody.Error()
			}
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		userAgent := request.Header.Get("User-Agent")
		values.DataLog.MapLog["user_agent"] = userAgent
		ipUser := ip.GetIP(request)
		values.DataLog.MapLog["ip"] = ipUser
		allFlag := request.URL.Query().Get("all")
		values.DataLog.MapLog["all"] = allFlag
		if !h.helperRateLimiting(request.Context(), body.RefreshJwt, writer) {
			return
		}
		h.ServiceAuth.Logout(request.Context(), body.RefreshJwt, userAgent, ipUser, allFlag)
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (h *HandlerAuth) helperRateLimiting(ctxRequest context.Context, keyLimiter string, writer http.ResponseWriter) bool {
	resp := &response.Response{
		Error: make(map[string]string),
	}
	counterRequest, headers, errRateLimiting := h.Limiter.RateLimiter(ctxRequest, keyLimiter)
	if errRateLimiting != nil {
		resp.Error["global"] = sherrors.ErrCriticalServer.Error()
		h.ResponseSend(writer, resp, http.StatusInternalServerError)
		return false
	}
	for key, value := range headers {
		writer.Header().Set(key, value)
	}
	if counterRequest > ratelimit.RequestLimit {
		writer.Header().Set("Retry-After", "60")
		resp.Error["rate_limit"] = sherrors.ErrRateLimiting.Error()
		h.ResponseSend(writer, resp, http.StatusTooManyRequests)
		return false
	}
	return true
}
