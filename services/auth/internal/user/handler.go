package user

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/common"
	"app/auth-service/internal/ip"
	"app/auth-service/internal/middleware"
	"errors"
	"net/http"
	"shared/loggers"
	"shared/requestutil"
	"shared/response"
	"shared/sherrors"
	"shared/shmiddleware"

	"github.com/go-playground/validator/v10"
)

type HandlerUser struct {
	*response.HandlerResponse
	*loggers.Logger
	Validate *validator.Validate
	*ServiceUser
}

func NewHandlerUser(router *http.ServeMux, service *ServiceUser, handlerResponse *response.HandlerResponse, validate *validator.Validate, logger *loggers.Logger, mv *middleware.ManagerMiddleware, sharedMv *shmiddleware.ManagerSharedMiddleware) {
	user := HandlerUser{
		ServiceUser:     service,
		HandlerResponse: handlerResponse,
		Logger:          logger,
		Validate:        validate,
	}
	chainMv := shmiddleware.Chain(
		sharedMv.HandlerAccessToken,
		sharedMv.RateLimiting,
	)
	router.Handle("PATCH /api/v1/user", chainMv(user.UpdateUser()))
	router.Handle("GET /api/v1/user", chainMv(user.GetUser()))
	router.Handle("DELETE /api/v1/user", chainMv(user.RemoveUser()))
	router.Handle("POST /api/v1/user/confirm", chainMv(mv.HandlerSessionToken(user.ConfirmUser())))
}

// UpdateUser godoc
// @Summary      Update user profile data or trigger sensitive update workflow
// @Description  Updates public info directly (returns 200 OK). If updating sensitive info (email/password), requires current password and triggers 2FA workflow (returns 202 Accepted).
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <access_token>"
// @Param        request        body      RequestUpdateUser  true  "Profile update payload"
// @Success      200      {object}  response.Response{data=model.Users,errors=nil} "Public profile details successfully updated instantly"
// @Success      202      {object}  response.Response{data=common.ResponseAuth,errors=nil} "Sensitive update accepted, 2FA confirmation workflow initiated"
// @Failure      400      {object}  response.NegativeResponse "Validation or business logic errors. Format: { \"errors\": { \"new_name\": \"new name must be between 2 and 64 characters\" } } or { \"errors\": { \"new_password\": \"the new_password cannot  contain email\" } } or { \"errors\": { \"email\": \"either a new email or an old one must be present\" } }"
// @Failure      401      {object}  response.NegativeResponse "Authentication errors. Format: { \"errors\": { \"auth\": \"incorrect password or email\" } } or { \"errors\": { \"auth\": \"invalid access token\" } }"
// @Failure      404      {object}  response.NegativeResponse "Data errors. Format: { \"errors\": { \"user\": \"not found user\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to update user\" } } or { \"errors\": { \"global\": \"failed to ensure security\" } }"
// @Router       /user [patch]
func (h *HandlerUser) UpdateUser() http.HandlerFunc {
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
		body, errBody := requestutil.HandlerRequest[RequestUpdateUser](request.Body, h.Validate)
		if errBody != nil {
			mapError := sherrors.MapError{Map: make(map[string]string, 5)}
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					switch {
					case err.Field() == "Email":
						values.DataLog.MapLog["email"] = body.Email
						mapError.Map["email"] = apperrors.ErrIncorrectEmail.Error()
					case err.Field() == "NewEmail":
						values.DataLog.MapLog["new_email"] = body.NewEmail
						mapError.Map["new_email"] = ErrIncorrectNewEmail.Error()
					case err.Field() == "NewName":
						values.DataLog.MapLog["new_name"] = body.NewName
						mapError.Map["new_name"] = ErrIncorrectNewName.Error()
					case err.Field() == "NewPassword":
						mapError.Map["new_password"] = apperrors.ErrIncorrectEnterNewPassword.Error()
					case err.Field() == "Password":
						mapError.Map["password"] = apperrors.ErrIncorrectEnterPassword.Error()
					}
				}
			} else {
				mapError.Map["body"] = errBody.Error()
			}
			resp.Error = mapError.Map
			values.DataLog.Errors = mapError.Error()
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		userUpdate, respAuth, errUpdateAuth := h.ServiceUser.UpdateUser(request.Context(), values.DataAuth.UserUUID, body)
		if errUpdateAuth != nil {
			values.DataLog.Errors = errUpdateAuth.Error()
			switch {
			case errors.Is(errUpdateAuth, apperrors.ErrNotFoundUser):
				resp.Error["user"] = errUpdateAuth.Error()
				h.ResponseSend(writer, resp, http.StatusNotFound)
			case errors.Is(errUpdateAuth, ErrNewPasswordContainEmail), errors.Is(errUpdateAuth, apperrors.ErrPasswordIsNotStrong):
				resp.Error["new_password"] = errUpdateAuth.Error()
				h.ResponseSend(writer, resp, http.StatusBadRequest)
			case errors.Is(errUpdateAuth, apperrors.ErrIncorrectPasswordOrEmail):
				resp.Error["auth"] = errUpdateAuth.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			case errors.Is(errUpdateAuth, ErrIncorrectChoiceEmail):
				resp.Error["email"] = errUpdateAuth.Error()
				h.ResponseSend(writer, resp, http.StatusBadRequest)
			default:
				resp.Error["global"] = errUpdateAuth.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		if respAuth != nil {
			resp.Data = respAuth
			h.ResponseSend(writer, resp, http.StatusAccepted)
			return
		}
		if userUpdate != nil {
			resp.Data = userUpdate
			h.ResponseSend(writer, resp, http.StatusOK)
			return
		}
	}
}

// GetUser godoc
// @Summary      Get user profile
// @Description  Retrieves current authenticated user's profile details from MySQL by UUID stored in appjwt access token.
// @Tags         user
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <access_token>"
// @Success      200      {object}  response.Response{data=ResponseUser,errors=nil} "User profile details successfully retrieved"
// @Failure      401      {object}  response.NegativeResponse "Token errors. Format: { \"errors\": { \"auth\": \"invalid access token\" } } or { \"errors\": { \"auth\": \"access token has expired\" } }"
// @Failure      404      {object}  response.NegativeResponse "Data errors. Format: { \"errors\": { \"user\": \"not found user\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to get user\" } }"
// @Router       /user [get]
func (h *HandlerUser) GetUser() http.HandlerFunc {
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
		respUser, errGetUser := h.ServiceUser.GetUser(request.Context(), values.DataAuth.UserUUID)
		if errGetUser != nil {
			values.DataLog.Errors = errGetUser.Error()
			if errors.Is(errGetUser, apperrors.ErrNotFoundUser) {
				resp.Error["user"] = errGetUser.Error()
				h.ResponseSend(writer, resp, http.StatusNotFound)
			} else {
				resp.Error["global"] = errGetUser.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respUser
		h.ResponseSend(writer, resp, http.StatusOK)
	}
}

// RemoveUser godoc
// @Summary      Initiate account deletion workflow
// @Description  Validates user credentials and the deletion type. If criteria are met, generates a confirmation session in Redis and triggers 2FA workflow (returns 202 Accepted).
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <access_token>"
// @Param        type           query     string  true  "Type of deletion: soft-delete or hard-delete"
// @Param        request        body      RequestRemoveUser  true  "Account credentials payload"
// @Success      202      {object}  response.Response{data=common.ResponseAuth,errors=nil} "Deletion request accepted, 2FA confirmation workflow initiated"
// @Failure      400      {object}  response.NegativeResponse "Client validation errors. Format: { \"errors\": { \"email\": \"incorrect email\" } } or { \"errors\": { \"action\": \"the type  must be a soft-delete or hard-delete\" } }"
// @Failure      401      {object}  response.NegativeResponse "Authentication errors. Format: { \"errors\": { \"auth\": \"incorrect password or email\" } } or { \"errors\": { \"auth\": \"invalid access token\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"critical error on the server side\" } } or { \"errors\": { \"global\": \"failed to remove user\" } }"
// @Router       /user [delete]
func (h *HandlerUser) RemoveUser() http.HandlerFunc {
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
		typeRemove := request.URL.Query().Get("type")
		values.DataLog.MapLog["type"] = typeRemove
		body, errBody := requestutil.HandlerRequest[RequestRemoveUser](request.Body, h.Validate)
		if errBody != nil {
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					if err.Field() == "Email" {
						values.DataLog.MapLog["email"] = body.Email
						values.DataLog.Errors = apperrors.ErrIncorrectEmail.Error()
						resp.Error["email"] = apperrors.ErrIncorrectEmail.Error()
					}
					if err.Field() == "Password" {
						values.DataLog.Errors = apperrors.ErrIncorrectEnterPassword.Error()
						resp.Error["password"] = apperrors.ErrIncorrectEnterPassword.Error()
					}
				}
			} else {
				values.DataLog.Errors = errBody.Error()
				resp.Error["body"] = errBody.Error()
			}
			h.ResponseSend(writer, resp, http.StatusBadRequest)
			return
		}
		respAuth, errRemoveAuth := h.ServiceUser.RemoveUser(request.Context(), body, typeRemove)
		if errRemoveAuth != nil {
			values.DataLog.Errors = errRemoveAuth.Error()
			switch {
			case errors.Is(errRemoveAuth, apperrors.ErrIncorrectPasswordOrEmail):
				resp.Error["auth"] = errRemoveAuth.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			case errors.Is(errRemoveAuth, sherrors.ErrIncorrectTypeRemove):
				resp.Error["action"] = errRemoveAuth.Error()
				h.ResponseSend(writer, resp, http.StatusBadRequest)
			default:
				resp.Error["global"] = errRemoveAuth.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			}
			return
		}
		resp.Success = true
		resp.Data = respAuth
		h.ResponseSend(writer, resp, http.StatusAccepted)
	}
}

// ConfirmUser godoc
// @Summary      Confirm 2FA code for profile updates or account deletion
// @Description  Validates the 6-digit OTP code from Redis session. For updates, applies changes to MySQL and returns user data (200 OK). For hard/soft deletion, purges sessions, publishes to Kafka, and returns no content (204 No Content).
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <access_token>"
// @Param        action         query     string  true  "Workflow action: update, soft-delete, or hard-delete"
// @Param        request        body      RequestConfirm  true  "Confirmation payload storing 6-digit code"
// @Success      200      {object}  response.Response{data=ResponseUser,errors=nil} "Sensitive profile details successfully updated"
// @Success      204      "Account successfully deleted (soft or hard), no content returned"
// @Failure      400      {object}  response.NegativeResponse "Client validation or workflow errors. Format: { \"errors\": { \"auth\": \"the code must be 6 characters\" } } or { \"errors\": { \"action\": \"the action must be update, soft-delete or hard-delete\" } }"
// @Failure      401      {object}  response.NegativeResponse "Authentication or session errors. Format: { \"errors\": { \"auth\": \"invalid access token\" } } or { \"errors\": { \"auth\": \"refresh token has expired\" } } or { \"errors\": { \"auth\": \"incorrect refresh token\" } } or { \"errors\": { \"auth\": \"refresh token renewal error\" } } or { \"errors\": { \"auth\": \"incorrect session id\" } } or { \"errors\": { \"auth\": \"authorization session has expired or does not exist\" } } or { \"errors\": { \"auth\": \"incorrect code\" } }"
// @Failure      404      {object}  response.NegativeResponse "Data errors. Format: { \"errors\": { \"user\": \"not found user\" } }"
// @Failure      429      {object}  response.NegativeResponse "Too many requests. Format: { \"errors\": { \"global\": \"the limit for sending requests per minute has been exceeded\" } }"
// @Failure      500      {object}  response.NegativeResponse "Server errors. Format: { \"errors\": { \"global\": \"failed to update user\" } } or { \"errors\": { \"global\": \"failed to delete user\" } } or { \"errors\": { \"global\": \"failed to remove user\" } } or { \"errors\": { \"global\": \"critical error on the server side\" } }"
// @Router       /user/confirm [post]
func (h *HandlerUser) ConfirmUser() http.HandlerFunc {
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
			if errValidate, okErrValidate := errBody.(validator.ValidationErrors); okErrValidate {
				for _, err := range errValidate {
					if err.Field() == "Code" {
						values.DataLog.Errors = apperrors.ErrIncorrectFormatCode.Error()
						resp.Error["auth"] = apperrors.ErrIncorrectFormatCode.Error()
					} else if err.Field() == "RefreshJwt" {
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
		action := request.URL.Query().Get("action")
		values.DataLog.MapLog["action"] = action
		userAgent := request.Header.Get("User-Agent")
		values.DataLog.MapLog["user_agent"] = userAgent
		ipUser := ip.GetIP(request)
		values.DataLog.MapLog["ip"] = ipUser
		respUserUpdate, errConfirm := h.ServiceUser.ConfirmUser(&ConfirmUserParams{
			CtxRequest: request.Context(),
			CodeUser:   body.Code,
			UserUUID:   values.DataAuth.UserUUID,
			SessionID:  values.DataAuth.SessionID,
			Action:     action,
			UserAgent:  userAgent,
			RefreshJWT: body.RefreshJwt,
			IP:         ipUser,
		})
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
			case errors.Is(errConfirm, apperrors.ErrNotFoundUser):
				resp.Error["user"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusNotFound)
			case errors.Is(errConfirm, ErrFailedRemoveUser), errors.Is(errConfirm, ErrFailedDeleteUser), errors.Is(errConfirm, ErrFailedUpdateUser), errors.Is(errConfirm, sherrors.ErrCriticalServer):
				resp.Error["global"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusInternalServerError)
			default:
				resp.Error["auth"] = errConfirm.Error()
				h.ResponseSend(writer, resp, http.StatusUnauthorized)
			}
			return
		}
		if respUserUpdate != nil {
			resp.Success = true
			resp.Data = respUserUpdate
			h.ResponseSend(writer, resp, http.StatusOK)
		} else {
			writer.WriteHeader(http.StatusNoContent)
		}
	}
}
