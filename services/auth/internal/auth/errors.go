package auth

import "errors"

var (
	ErrUserAlreadyExist        = errors.New("user already exist")
	ErrIncorrectAction         = errors.New("the action must be register, login or recovery")
	ErrCreateUser              = errors.New("failed to create user")
	ErrIncorrectName           = errors.New("name must be between 2 and 64 characters")
	ErrFailedRecoveryUser      = errors.New("failed to recovery user")
	ErrNotSpecifiedNewPassword = errors.New("new password not specified")
	ErrIncorrectActionRecovery = errors.New("action must be recovery_password or recovery_user")
	ErrChangePassword          = errors.New("failed to change password")
	ErrPasswordEmpty           = errors.New("password is empty")
	ErrRefreshExpired          = errors.New("session has expired try again later")
)
