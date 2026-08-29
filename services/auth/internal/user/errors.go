package user

import "errors"

var (
	ErrFailedGetUser           = errors.New("failed to get user")
	ErrFailedUpdateUser        = errors.New("failed to update user")
	ErrIncorrectChoiceEmail    = errors.New("either a new email or an old one must be present")
	ErrIncorrectNewEmail       = errors.New("incorrect new email")
	ErrIncorrectNewName        = errors.New("new name must be between 2 and 64 characters")
	ErrIncorrectAction         = errors.New("the action must be update, soft-delete or hard-delete")
	ErrFailedDeleteUser        = errors.New("failed to delete user")
	ErrFailedRemoveUser        = errors.New("failed to remove user")
	ErrNewPasswordContainEmail = errors.New("the new_password cannot contain your email")
)
