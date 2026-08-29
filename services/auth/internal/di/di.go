package di

import (
	"app/auth-service/internal/common"
	"app/auth-service/internal/model"
	"context"
)

type IRepoUser interface {
	CreateUser(ctxRequest context.Context, user *model.Users) error
	UserExistsByEmail(ctxRequest context.Context, email string) (bool, error)
	GetPasswordByEmail(ctxRequest context.Context, email string) (string, error)
	GetUserUUIDByEmail(ctxRequest context.Context, email string) (string, error)
	RecoveryUser(ctxRequest context.Context, userUUID string) error
	GetEmailByUUID(ctxRequest context.Context, userUUID string) (string, error)
	UpdateUser(ctxRequest context.Context, user *model.Users, userUUID string) error
	GetUserByEmail(ctxRequest context.Context, email string) (*model.Users, error)
	DeleteUsersByTimer() ([]string, error)
}
type IServiceAuth interface {
	HelperAuth(ctxRequest context.Context, action string, dataUser map[string]any) (*common.ResponseAuth, error)
	HelperSecurity(ctxRequest context.Context, oldUserAgent, newUserAgent, oldIP, newIP, userUUID, refreshUUID, email string) error
}
type IRepoAuth interface {
	DeleteUserRefreshes(ctxRequest context.Context, userUUID string) error
	LogoutRefresh(ctxRequest context.Context, userUUID, refreshKey string) error
	GetUserSession(ctxRequest context.Context, sessionID, action string) (map[string]string, error)
	GetRefreshData(ctxRequest context.Context, userUUID, refreshUUID string) (*common.RefreshData, string, error)
}
