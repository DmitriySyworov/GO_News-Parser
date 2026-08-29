package auth

import (
	authconfig "app/auth-service/config"
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/appjwt"
	"app/auth-service/internal/appuseragent"
	"app/auth-service/internal/code"
	"app/auth-service/internal/common"
	"app/auth-service/internal/di"
	"app/auth-service/internal/ip"
	"app/auth-service/internal/model"
	"app/auth-service/internal/notifer"
	"app/auth-service/internal/password"
	"context"
	"shared/loggers"
	"shared/sherrors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ServiceAuth struct {
	Repo      IRepositoryRedis
	IRepoUser di.IRepoUser
	Conf      *authconfig.Config
	Notifer   *notifer.Notifer
	Logger    *loggers.Logger
}

func NewServiceAuth(repo IRepositoryRedis, nt *notifer.Notifer, repoUser di.IRepoUser, conf *authconfig.Config, logger *loggers.Logger) *ServiceAuth {
	return &ServiceAuth{
		Repo:      repo,
		IRepoUser: repoUser,
		Conf:      conf,
		Notifer:   nt,
		Logger:    logger,
	}
}

func (s *ServiceAuth) Register(ctxRequest context.Context, body *RequestRegister) (*common.ResponseAuth, error) {
	mapError := sherrors.MapError{Map: make(map[string]string, 2)}
	errValidatePassword := password.ValidatePassword(body.Password, body.Email, []string{body.Name})
	if errValidatePassword != nil {
		mapError.Map["password"] = errValidatePassword.Error()
	}
	userExist, errCheckUser := s.IRepoUser.UserExistsByEmail(ctxRequest, body.Email)
	if errCheckUser != nil {
		mapError.Map["global"] = sherrors.ErrCriticalServer.Error()
	} else if userExist {
		mapError.Map["user"] = ErrUserAlreadyExist.Error()
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	hashPassword, errGeneratePassword := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if errGeneratePassword != nil {
		s.Logger.Error("failed to hash the password")
		return nil, apperrors.ErrFailedSecurity
	}
	const sizeRegisterMap = 4
	dataMap := make(map[string]any, sizeRegisterMap)
	dataMap[nameKey] = body.Name
	dataMap[passwordKey] = string(hashPassword)
	dataMap[emailKey] = body.Email
	respAuth, errAuth := s.HelperAuth(ctxRequest, ActionRegister, dataMap)
	if errAuth != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return respAuth, nil
}
func (s *ServiceAuth) Login(ctxRequest context.Context, body *RequestLogin) (*common.ResponseAuth, error) {
	hashPassword, errGetPassword := s.IRepoUser.GetPasswordByEmail(ctxRequest, body.Email)
	if errGetPassword != nil {
		return nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	if bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(body.Password)) != nil {
		return nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	const sizeLoginMap = 2
	dataMap := make(map[string]any, sizeLoginMap)
	dataMap[emailKey] = body.Email
	respAuth, errAuth := s.HelperAuth(ctxRequest, ActionLogin, dataMap)
	if errAuth != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return respAuth, nil
}
func (s *ServiceAuth) Recovery(ctxRequest context.Context, body *RequestRecovery, action string) (*common.ResponseAuth, error) {
	mapError := &sherrors.MapError{
		Map: make(map[string]string, 2),
	}
	hashedPassword, errGetPassword := s.IRepoUser.GetPasswordByEmail(ctxRequest, body.Email)
	if errGetPassword != nil {
		mapError.Map["email"] = apperrors.ErrNotFoundUser.Error()
	}
	if action != ActionRecoveryPassword && action != ActionRecoveryUser {
		mapError.Map["action"] = ErrIncorrectActionRecovery.Error()
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	if action == ActionRecoveryUser {
		if body.Password == "" {
			return nil, ErrPasswordEmpty
		}
		if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(body.Password)) != nil {
			return nil, apperrors.ErrIncorrectPasswordOrEmail
		}
	}
	const sizeRecoveryMap = 2
	dataMap := make(map[string]any, sizeRecoveryMap)
	dataMap[emailKey] = body.Email
	respAuth, errAuth := s.HelperAuth(ctxRequest, action, dataMap)
	if errAuth != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return respAuth, nil
}
func (s *ServiceAuth) HelperAuth(ctxRequest context.Context, action string, dataUser map[string]any) (*common.ResponseAuth, error) {
	sessionID := uuid.New().String()
	codeAuth, errCode := code.GenerateCode()
	if errCode != nil {
		s.Logger.Error("failed create code: ", errCode)
		return nil, apperrors.ErrFailedSecurity
	}
	codeAuthStr := strconv.Itoa(codeAuth)
	emailUser := dataUser[emailKey].(string)
	dataUser[common.CodeKey] = codeAuthStr
	if s.Repo.CreateUserSession(ctxRequest, sessionID, action, dataUser) != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	j := appjwt.NewJWT(s.Conf.Signature, s.Logger)
	token, errJwtSession := j.CreateSessionJWT(sessionID)
	if errJwtSession != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	go s.Notifer.HelperSendEmailEvent(&notifer.NotificationEvent{
		LetterAuth: &notifer.LetterAuth{
			Code:      codeAuthStr,
			ValidTime: time.Now().Add(common.TTLSessionJWT).Unix(),
		},
		EmailTo:   emailUser,
		EventUUID: sessionID,
	}, notifer.SendCodeAction)
	return &common.ResponseAuth{
		Message:    "we have sent a confirmation code to the following email address: " + dataUser[emailKey].(string),
		SessionJwt: token,
	}, nil
}

const (
	ActionRegister         = "register"
	ActionLogin            = "login"
	ActionRecoveryUser     = "recovery_user"
	ActionRecoveryPassword = "recovery_password"

	nameKey     = "name"
	emailKey    = "email"
	passwordKey = "password"
)

func (s *ServiceAuth) Confirm(ctxRequest context.Context, body *RequestConfirm, sessionID, action, userAgent, ipUser string) (*ResponseConfirm, error) {
	mapError := sherrors.MapError{Map: make(map[string]string, 2)}
	if len(sessionID) != 36 {
		mapError.Map["session"] = apperrors.ErrIncorrectSessionID.Error()
	}
	if action != ActionRecoveryUser && action != ActionRecoveryPassword && action != ActionLogin && action != ActionRegister {
		mapError.Map["action"] = ErrIncorrectAction.Error()
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	if action == ActionRecoveryPassword && body.NewPassword == "" {
		return nil, ErrNotSpecifiedNewPassword
	}
	dataUser, errGetDataSession := s.Repo.GetUserSession(ctxRequest, sessionID, action)
	if errGetDataSession != nil {
		return nil, apperrors.ErrSessionExpired
	}
	if strconv.Itoa(body.Code) != dataUser[common.CodeKey] {
		mapError.Map["code"] = apperrors.ErrIncorrectCode.Error()
		mapError.Map[common.AttemptsLeft] = dataUser[common.AttemptsLeft]
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	var userUUID string
	email := dataUser[emailKey]
	switch action {
	case ActionRegister:
		userExist, errCheckUser := s.IRepoUser.UserExistsByEmail(ctxRequest, email)
		if errCheckUser != nil {
			return nil, sherrors.ErrCriticalServer
		}
		if userExist {
			return nil, ErrUserAlreadyExist
		}
		userUUID = uuid.New().String()
		if s.IRepoUser.CreateUser(ctxRequest, &model.Users{
			Name:     dataUser[nameKey],
			Email:    email,
			Password: dataUser[passwordKey],
			UserUUID: userUUID,
		}) != nil {
			return nil, ErrCreateUser
		}
	case ActionLogin:
		if uUUID, errGetUserUUID := s.IRepoUser.GetUserUUIDByEmail(ctxRequest, email); errGetUserUUID != nil {
			return nil, apperrors.ErrNotFoundUser
		} else {
			userUUID = uUUID
		}
	case ActionRecoveryUser:
		if uUUID, errGetUserUUID := s.IRepoUser.GetUserUUIDByEmail(ctxRequest, email); errGetUserUUID != nil {
			return nil, apperrors.ErrNotFoundUser
		} else {
			userUUID = uUUID
			if s.IRepoUser.RecoveryUser(ctxRequest, uUUID) != nil {
				return nil, ErrFailedRecoveryUser
			}
		}
	case ActionRecoveryPassword:
		user, errGetUser := s.IRepoUser.GetUserByEmail(ctxRequest, email)
		if errGetUser != nil {
			return nil, apperrors.ErrNotFoundUser
		}
		userUUID = user.UserUUID
		hashPassword, errHashPass := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if errHashPass != nil {
			s.Logger.Error("failed to hashed password: " + errHashPass.Error())
			return nil, apperrors.ErrFailedSecurity
		}
		user.Password = string(hashPassword)
		if s.IRepoUser.UpdateUser(ctxRequest, user, user.UserUUID) != nil {
			return nil, ErrChangePassword
		}
	}
	refreshUUID := uuid.New().String()
	respConfirm, errCreateToken := s.helperCreateToken(userUUID, refreshUUID)
	if errCreateToken != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	if s.Repo.CreateRefresh(ctxRequest, &CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: refreshUUID,
		UserAgent:   userAgent,
		IPUser:      ipUser,
		EmailUser:   email,
	}) != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return respConfirm, nil
}
func (s *ServiceAuth) Refresh(ctxRequest context.Context, oldRefreshToken, userAgent, ipUser string) (*ResponseConfirm, error) {
	j := appjwt.NewJWT(s.Conf.Signature, s.Logger)
	refreshToken, errParseRefresh := j.ParseRefreshToken(oldRefreshToken)
	if errParseRefresh != nil {
		return nil, errParseRefresh
	}
	userUUID := refreshToken.UserUUID
	refreshData, oldRefreshStrKey, errGetRefreshData := s.Repo.GetRefreshData(ctxRequest, userUUID, refreshToken.RefreshUUID)
	if errGetRefreshData != nil {
		return nil, apperrors.ErrRenewalRefresh
	}
	if errSecurity := s.HelperSecurity(ctxRequest, refreshData.UserAgent, userAgent, refreshData.IP, ipUser, userUUID, refreshToken.RefreshUUID, refreshData.Email); errSecurity != nil {
		return nil, errSecurity
	}
	newRefreshUUID := uuid.New().String()
	respConfirm, errConfirm := s.helperCreateToken(userUUID, newRefreshUUID)
	if errConfirm != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	newRefreshKey := newRefreshUUID + NullByte + userAgent + NullByte + ipUser + NullByte + refreshData.Email
	if errRotation := s.Repo.RotationRefresh(ctxRequest, userUUID, newRefreshKey, oldRefreshStrKey); errRotation != nil {
		return nil, apperrors.ErrRenewalRefresh
	}
	return respConfirm, nil
}
func (s *ServiceAuth) Logout(ctxRequest context.Context, refreshToken, userAgent, ipUser, allFlag string) {
	j := appjwt.NewJWT(s.Conf.Signature, s.Logger)
	refreshTokenData, errParseRefresh := j.ParseRefreshToken(refreshToken)
	if errParseRefresh != nil {
		return
	}
	userUUID := refreshTokenData.UserUUID
	switch allFlag {
	case "true":
		if s.Repo.DeleteUserRefreshes(ctxRequest, userUUID) != nil {
			return
		}
	default:
		refreshData, refreshByteKey, errGetRefreshKey := s.Repo.GetRefreshData(ctxRequest, userUUID, refreshTokenData.RefreshUUID)
		if errGetRefreshKey != nil {
			return
		}
		if errSecurity := s.HelperSecurity(ctxRequest, refreshData.UserAgent, userAgent, refreshData.IP, ipUser, userUUID, refreshTokenData.RefreshUUID, refreshData.Email); errSecurity != nil {
			return
		}
		if s.Repo.LogoutRefresh(ctxRequest, userUUID, refreshByteKey) != nil {
			return
		}
	}
}
func (s *ServiceAuth) HelperSecurity(ctxRequest context.Context, oldUserAgent, newUserAgent, oldIP, newIP, userUUID, refreshUUID, email string) error {
	matchUserAgent, newDevice := appuseragent.ValidateUserAgent(oldUserAgent, newUserAgent)
	if !matchUserAgent && !ip.CompareIP(oldIP, newIP) {
		s.Logger.Error("SECURITY ALERT: User-Agent mismatch for user " + userUUID + "! Force logout triggered. Expected User-Agent: " + oldUserAgent + " , Got: " + newUserAgent)
		if errDeleteRefreshes := s.Repo.DeleteUserRefreshes(ctxRequest, userUUID); errDeleteRefreshes != nil {
			s.Logger.Error("failed to force logout user_uuid: " + userUUID + " refresh_uuid: " + refreshUUID)
		}
		go s.Notifer.HelperSendEmailEvent(&notifer.NotificationEvent{
			LetterSecurity: &notifer.LetterSecurity{
				Device: newDevice,
				IP:     newIP,
			},
			EmailTo:   email,
			EventUUID: refreshUUID,
		}, notifer.SendSecurityAlertAction)
		return apperrors.ErrRenewalRefresh
	}
	if !matchUserAgent {
		go s.Notifer.HelperSendEmailEvent(&notifer.NotificationEvent{
			LetterSecurity: &notifer.LetterSecurity{
				Device: newDevice,
				IP:     newIP,
			},
			EmailTo:   email,
			EventUUID: refreshUUID,
		}, notifer.SendNewDeviceAction)
	}
	return nil
}
func (s *ServiceAuth) helperCreateToken(userUUID, refreshUUID string) (*ResponseConfirm, error) {
	j := appjwt.NewJWT(s.Conf.Signature, s.Logger)
	accessJwt, errCreateAccess := j.CreateAccessJWT(userUUID)
	if errCreateAccess != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	refreshJwt, errCreateRefresh := j.CreateRefreshJWT(userUUID, refreshUUID)
	if errCreateRefresh != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return &ResponseConfirm{
		AccessJwt:  accessJwt,
		RefreshJwt: refreshJwt,
	}, nil
}
