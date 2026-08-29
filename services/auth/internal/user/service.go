package user

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/appjwt"
	"app/auth-service/internal/common"
	"app/auth-service/internal/di"
	"app/auth-service/internal/model"
	"app/auth-service/internal/password"
	"context"
	"errors"
	"shared/loggers"
	"shared/shconstant"
	"shared/sherrors"
	"shared/shkafka"
	"shared/shprotos/event"
	"strconv"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/proto"
)

type ServiceUser struct {
	Repo IRepositoryUser
	di.IServiceAuth
	di.IRepoAuth
	Producer  *shkafka.KafkaProducer
	Logger    *loggers.Logger
	Signature string
}

func NewServiceUser(repo IRepositoryUser, iServiceAuth di.IServiceAuth, iRepoAuth di.IRepoAuth, producer *shkafka.KafkaProducer, signature string, logger *loggers.Logger) *ServiceUser {
	return &ServiceUser{
		Repo:         repo,
		IServiceAuth: iServiceAuth,
		IRepoAuth:    iRepoAuth,
		Producer:     producer,
		Logger:       logger,
		Signature:    signature,
	}
}

func (s *ServiceUser) UpdateUser(ctxRequest context.Context, userUUID string, body *RequestUpdateUser) (*model.Users, *common.ResponseAuth, error) {
	if body.NewPassword == "" && body.NewEmail == "" {
		user, errGet := s.Repo.GetUserByUUID(ctxRequest, userUUID)
		if errGet != nil || user == nil {
			return nil, nil, apperrors.ErrNotFoundUser
		}
		user.Name = body.NewName
		if s.Repo.UpdateUser(ctxRequest, user, userUUID) != nil {
			return nil, nil, ErrFailedUpdateUser
		}
		return user, nil, nil
	}
	user, errGetUser := s.Repo.GetUserByUUID(ctxRequest, userUUID)
	if errGetUser != nil {
		return nil, nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
		return nil, nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	var newHashedPassword string
	if body.NewPassword != "" {
		if body.NewEmail != "" {
			if errNewPassword := password.ValidatePassword(body.NewPassword, body.NewEmail, []string{body.NewName, user.Name}); errNewPassword != nil {
				if errors.Is(errNewPassword, apperrors.ErrPasswordIsNotStrong) {
					return nil, nil, errNewPassword
				}
				return nil, nil, ErrNewPasswordContainEmail
			}
		} else {
			if errNewPassword := password.ValidatePassword(body.NewPassword, user.Email, []string{body.NewName, user.Name}); errNewPassword != nil {
				if errors.Is(errNewPassword, apperrors.ErrPasswordIsNotStrong) {
					return nil, nil, errNewPassword
				}
				return nil, nil, ErrNewPasswordContainEmail
			}
		}
		hashPassword, errHash := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if errHash != nil {
			return nil, nil, apperrors.ErrFailedSecurity
		}
		newHashedPassword = string(hashPassword)
	}
	var sendEmail string
	if body.Email != "" && body.NewEmail == "" {
		sendEmail = body.Email
	} else if body.Email == "" && body.NewEmail != "" {
		sendEmail = body.NewEmail
	} else {
		return nil, nil, ErrIncorrectChoiceEmail
	}
	const sizeUpdateDataMap = 5
	dataUser := make(map[string]any, sizeUpdateDataMap)
	dataUser[emailKey] = sendEmail
	dataUser[newEmailKey] = body.NewEmail
	dataUser[newNameKey] = body.NewName
	dataUser[newPasswordKey] = newHashedPassword
	respAuth, errAuth := s.HelperAuth(ctxRequest, actionUpdate, dataUser)
	if errAuth != nil {
		return nil, nil, apperrors.ErrFailedSecurity
	}
	return nil, respAuth, nil
}
func (s *ServiceUser) GetUser(ctxRequest context.Context, userUUID string) (*ResponseUser, error) {
	user, errGet := s.Repo.GetResponseUserByUUID(ctxRequest, userUUID)
	if errGet != nil {
		return nil, errGet
	}
	return user, nil
}
func (s *ServiceUser) RemoveUser(ctxRequest context.Context, body *RequestRemoveUser, typeRemove string) (*common.ResponseAuth, error) {
	if typeRemove != shconstant.TypeSoftDelete && typeRemove != shconstant.TypeHardDelete {
		return nil, sherrors.ErrIncorrectTypeRemove
	}
	hashPassword, errGetPassword := s.Repo.GetPasswordByEmail(ctxRequest, body.Email)
	if errGetPassword != nil {
		return nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	if bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(body.Password)) != nil {
		return nil, apperrors.ErrIncorrectPasswordOrEmail
	}
	const sizeRemoveDataMap = 2
	dataUser := make(map[string]any, sizeRemoveDataMap)
	dataUser[emailKey] = body.Email
	respAuth, errAuth := s.HelperAuth(ctxRequest, typeRemove, dataUser)
	if errAuth != nil {
		return nil, apperrors.ErrFailedSecurity
	}
	return respAuth, nil
}

const (
	actionUpdate = "update"

	emailKey       = "email"
	newEmailKey    = "new_email"
	newNameKey     = "new_name"
	newPasswordKey = "new_password"
)

type ConfirmUserParams struct {
	CtxRequest context.Context
	CodeUser   int
	UserUUID   string
	SessionID  string
	Action     string
	UserAgent  string
	RefreshJWT string
	IP         string
}

func (s *ServiceUser) ConfirmUser(params *ConfirmUserParams) (*ResponseUser, error) {
	mapError := sherrors.MapError{Map: make(map[string]string, 2)}
	if len(params.SessionID) != 36 {
		mapError.Map["auth"] = apperrors.ErrIncorrectSessionID.Error()
	}
	if params.Action != shconstant.TypeHardDelete && params.Action != shconstant.TypeSoftDelete && params.Action != actionUpdate {
		mapError.Map["action"] = ErrIncorrectAction.Error()
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	j := appjwt.NewJWT(s.Signature, s.Logger)
	refreshToken, errParseRefresh := j.ParseRefreshToken(params.RefreshJWT)
	if errParseRefresh != nil {
		return nil, errParseRefresh
	}
	if refreshToken.UserUUID != params.UserUUID {
		s.Logger.Error("SECURITY ALERT: UserUUID mismatch! Force logout triggered. Access UserUUID: " + params.UserUUID + " , Refresh UserUUID: " + refreshToken.UserUUID)
		if errDelUserRefreshes := s.IRepoAuth.DeleteUserRefreshes(params.CtxRequest, refreshToken.UserUUID); errDelUserRefreshes != nil {
			s.Logger.Error("failed to delete user: " + refreshToken.UserUUID + " refreshes: " + errDelUserRefreshes.Error())
		}
		if errDelUserRefreshes := s.IRepoAuth.DeleteUserRefreshes(params.CtxRequest, params.UserUUID); errDelUserRefreshes != nil {
			s.Logger.Error("failed to delete user: " + params.UserUUID + " refreshes: " + errDelUserRefreshes.Error())
		}
		return nil, apperrors.ErrRenewalRefresh
	}
	refreshUUID := refreshToken.RefreshUUID
	refreshData, refreshByteKey, errGetRefreshData := s.IRepoAuth.GetRefreshData(params.CtxRequest, params.UserUUID, refreshUUID)
	if errGetRefreshData != nil {
		return nil, apperrors.ErrRenewalRefresh
	}
	if errSecurity := s.IServiceAuth.HelperSecurity(params.CtxRequest, refreshData.UserAgent, params.UserAgent, refreshData.IP, params.IP, params.UserUUID, refreshUUID, refreshData.Email); errSecurity != nil {
		return nil, errSecurity
	}
	dataSession, errGetDataSession := s.IRepoAuth.GetUserSession(params.CtxRequest, params.SessionID, params.Action)
	if errGetDataSession != nil {
		return nil, apperrors.ErrSessionExpired
	}
	if strconv.Itoa(params.CodeUser) != dataSession[common.CodeKey] {
		mapError.Map["code"] = apperrors.ErrIncorrectCode.Error()
		mapError.Map[common.AttemptsLeft] = dataSession[common.AttemptsLeft]
	}
	if len(mapError.Map) != 0 {
		return nil, mapError
	}
	switch params.Action {
	case actionUpdate:
		userExist, errCheckUser := s.Repo.UserExistsByUserUUID(params.CtxRequest, params.UserUUID)
		if errCheckUser != nil {
			return nil, sherrors.ErrCriticalServer
		}
		if !userExist {
			return nil, apperrors.ErrNotFoundUser
		}
		user := &model.Users{
			Name:     dataSession[newNameKey],
			Email:    dataSession[newEmailKey],
			Password: dataSession[newPasswordKey],
		}
		if s.Repo.UpdateUser(params.CtxRequest, user, params.UserUUID) != nil {
			return nil, ErrFailedUpdateUser
		}
		return &ResponseUser{
			CreatedAt: user.CreatedAt.Format(time.DateOnly),
			UpdatedAt: user.UpdatedAt.Format(time.DateOnly),
			Name:      user.Name,
			Email:     user.Email,
			UserUUID:  user.UserUUID,
		}, nil
	case shconstant.TypeHardDelete:
		dataEvent, errMarshalEvent := proto.Marshal(&event.DeleteUserDataEvent{
			UserUuidList: []string{params.UserUUID},
			EventUuid:    uuid.New().String(),
		})
		if errMarshalEvent != nil {
			s.Logger.Error("failed to marshal proto event: " + errMarshalEvent.Error())
			return nil, ErrFailedDeleteUser
		}
		if s.Repo.DeleteUser(params.CtxRequest, params.UserUUID) != nil {
			return nil, ErrFailedDeleteUser
		}
		if errDelUserRefreshes := s.IRepoAuth.DeleteUserRefreshes(params.CtxRequest, params.UserUUID); errDelUserRefreshes != nil {
			s.Logger.Error("failed to delete user: " + params.UserUUID + " refreshes: " + errDelUserRefreshes.Error())
		}
		go func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutSendEventKafka)
			defer cancel()
			if errSendEvent := s.Producer.SendInstantEvent(ctxTimeout, params.UserUUID, dataEvent); errSendEvent != nil {
				s.Logger.Error("failed to send event deleted_user: " + errSendEvent.Error())
			} else {
				s.Logger.Info("event deleted_user successfully sent to Kafka in background")
			}
		}()
	case shconstant.TypeSoftDelete:
		if s.Repo.RemoveUser(params.CtxRequest, params.UserUUID) != nil {
			return nil, ErrFailedRemoveUser
		}
		if errDelRefreshKey := s.IRepoAuth.LogoutRefresh(params.CtxRequest, params.UserUUID, refreshByteKey); errDelRefreshKey != nil {
			s.Logger.Error("failed to delete refresh key: " + errDelRefreshKey.Error())
		}
	default:
		return nil, apperrors.ErrSessionExpired
	}
	return nil, nil
}
