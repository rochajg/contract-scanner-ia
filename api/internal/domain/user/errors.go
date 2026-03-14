package user

import "errors"

var (
	ErrEmailAlreadyExists    = errors.New("email already in use")
	ErrUsernameAlreadyExists = errors.New("username already in use")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidCredentials    = errors.New("invalid email or password")
)
