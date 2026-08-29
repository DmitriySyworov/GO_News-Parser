package user

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/model"
	"context"
	"shared/loggers"
	"shared/storage"

	"gorm.io/gorm/clause"
)

type IRepositoryUser interface {
	CreateUser(ctxRequest context.Context, user *model.Users) error
	UpdateUser(ctxRequest context.Context, user *model.Users, userUUID string) error
	UserExistsByEmail(ctxRequest context.Context, email string) (bool, error)
	UserExistsByUserUUID(ctxRequest context.Context, userUUID string) (bool, error)
	GetResponseUserByUUID(ctxRequest context.Context, userUUID string) (*ResponseUser, error)
	GetEmailByUUID(ctxRequest context.Context, userUUID string) (string, error)
	GetUserByUUID(ctxRequest context.Context, userUUID string) (*model.Users, error)
	GetUserByEmail(ctxRequest context.Context, email string) (*model.Users, error)
	GetPasswordByEmail(ctxRequest context.Context, email string) (string, error)
	GetUserUUIDByEmail(ctxRequest context.Context, email string) (string, error)
	RemoveUser(ctxRequest context.Context, userUUID string) error
	DeleteUser(ctxRequest context.Context, userUUID string) error
	RecoveryUser(ctxRequest context.Context, userUUID string) error
	DeleteUsersByTimer() ([]string, error)
}
type RepositoryUser struct {
	*storage.Postgres
	*loggers.Logger
}

func NewRepositoryUser(postgres *storage.Postgres, logger *loggers.Logger) IRepositoryUser {
	return &RepositoryUser{
		Postgres: postgres,
		Logger:   logger,
	}
}
func (r *RepositoryUser) CreateUser(ctxRequest context.Context, user *model.Users) error {
	if errCreate := r.Postgres.
		WithContext(ctxRequest).
		Create(user).Error; errCreate != nil {
		r.Logger.Error("failed to create user: ", errCreate)
		return errCreate
	}
	return nil
}
func (r *RepositoryUser) UpdateUser(ctxRequest context.Context, user *model.Users, userUUID string) error {
	if errUpdate := r.Postgres.
		WithContext(ctxRequest).
		Clauses(clause.Returning{}).
		Where("user_uuid = ?", userUUID).
		Updates(user).Error; errUpdate != nil {
		r.Logger.Error("failed to update user: ", errUpdate)
		return errUpdate
	}
	return nil
}
func (r *RepositoryUser) UserExistsByEmail(ctxRequest context.Context, email string) (bool, error) {
	var isExist bool
	if errCheckUser := r.Postgres.
		WithContext(ctxRequest).
		Raw(`SELECT EXISTS(
				 SELECT FROM users
				 WHERE email = ?)`, email).Scan(&isExist).Error; errCheckUser != nil {
		r.Logger.Error("failed to check if the user exists by email: ", errCheckUser)
		return false, errCheckUser
	}
	return isExist, nil
}
func (r *RepositoryUser) UserExistsByUserUUID(ctxRequest context.Context, userUUID string) (bool, error) {
	var isExist bool
	if errCheckUser := r.Postgres.
		WithContext(ctxRequest).
		Raw(`SELECT EXISTS(
				 SELECT FROM users
				 WHERE user_uuid = ?)`, userUUID).Scan(&isExist).Error; errCheckUser != nil {
		r.Logger.Error("failed to check if the user exists by user_uuid: ", errCheckUser)
		return false, errCheckUser
	}
	return isExist, nil
}
func (r *RepositoryUser) GetResponseUserByUUID(ctxRequest context.Context, userUUID string) (*ResponseUser, error) {
	user := &ResponseUser{}
	if errGet := r.Postgres.WithContext(ctxRequest).
		Raw(`SELECT created_at, updated_at, name, email, user_uuid FROM users
WHERE user_uuid = ? AND deleted_at IS NULL`, userUUID).Scan(user).Error; errGet != nil {
		r.Logger.Error("failed to get user: " + errGet.Error())
		return nil, errGet
	}
	if user.UserUUID == "" {
		return nil, apperrors.ErrNotFoundUser
	}
	return user, nil
}
func (r *RepositoryUser) GetUserByUUID(ctxRequest context.Context, userUUID string) (*model.Users, error) {
	user := &model.Users{}
	if errGet := r.Postgres.
		WithContext(ctxRequest).Where("user_uuid = ?", userUUID).Take(user).Error; errGet != nil {
		return nil, errGet
	}
	return user, nil
}

func (r *RepositoryUser) GetUserByEmail(ctxRequest context.Context, email string) (*model.Users, error) {
	user := &model.Users{}
	if errGet := r.Postgres.WithContext(ctxRequest).Where("email = ?", email).Take(user).Error; errGet != nil {
		return nil, errGet
	}
	return user, nil
}
func (r *RepositoryUser) GetPasswordByEmail(ctxRequest context.Context, email string) (string, error) {
	var password string
	if errGetPassword := r.Postgres.
		WithContext(ctxRequest).
		Raw(`SELECT password FROM users
                 WHERE email = ?`, email).Scan(&password).Error; errGetPassword != nil {
		r.Logger.Error("failed to get user password: ", errGetPassword)
		return "", ErrFailedGetUser
	}
	if password == "" {
		return "", apperrors.ErrNotFoundUser
	}
	return password, nil
}
func (r *RepositoryUser) GetEmailByUUID(ctxRequest context.Context, userUUID string) (string, error) {
	var email string
	if errGetEmail := r.Postgres.
		WithContext(ctxRequest).
		Raw(`SELECT email FROM users
                 WHERE user_uuid = ?`, userUUID).Scan(&email).Error; errGetEmail != nil {
		r.Logger.Error("failed to get user password: ", errGetEmail)
		return "", ErrFailedGetUser
	}
	if email == "" {
		return "", apperrors.ErrNotFoundUser
	}
	return email, nil
}
func (r *RepositoryUser) GetUserUUIDByEmail(ctxRequest context.Context, email string) (string, error) {
	var userUUID string
	if errGetUserUUID := r.Postgres.
		WithContext(ctxRequest).
		Raw(`SELECT user_uuid FROM users
			     WHERE email = ?`, email).Scan(&userUUID).Error; errGetUserUUID != nil {
		r.Logger.Error("failed to get userUUID: " + errGetUserUUID.Error())
		return "", ErrFailedGetUser
	}
	if userUUID == "" {
		return "", apperrors.ErrNotFoundUser
	}
	return userUUID, nil
}
func (r *RepositoryUser) RemoveUser(ctxRequest context.Context, userUUID string) error {
	if errRemove := r.Postgres.
		WithContext(ctxRequest).
		Where("user_uuid = ?", userUUID).Delete(&model.Users{}).Error; errRemove != nil {
		r.Logger.Error("failed to remove user: ", errRemove)
		return errRemove
	}
	return nil
}
func (r *RepositoryUser) DeleteUser(ctxRequest context.Context, userUUID string) error {
	if errDelete := r.Postgres.
		WithContext(ctxRequest).
		Unscoped().
		Where("user_uuid = ?", userUUID).
		Delete(&model.Users{}).Error; errDelete != nil {
		r.Logger.Error("failed to delete user: ", errDelete)
		return errDelete
	}
	return nil
}
func (r *RepositoryUser) RecoveryUser(ctxRequest context.Context, userUUID string) error {
	if errRecovery := r.Postgres.
		Model(&model.Users{}).
		WithContext(ctxRequest).
		Unscoped().
		Where("user_uuid = ?", userUUID).
		Update("deleted_at", nil).Error; errRecovery != nil {
		r.Logger.Error("failed to recovery user: " + errRecovery.Error())
		return errRecovery
	}
	return nil
}
func (r *RepositoryUser) DeleteUsersByTimer() ([]string, error) {
	var sliceDeleteUserUUID []string
	if errDelete := r.Postgres.Raw(`
			DELETE FROM users
			WHERE deleted_at <= now() - INTERVAL '30 days'
			RETURNING user_uuid
			`).Scan(&sliceDeleteUserUUID).
		Error; errDelete != nil {
		r.Logger.Error("failed to delete users by timer: " + errDelete.Error())
		return nil, errDelete
	}
	return sliceDeleteUserUUID, nil
}
