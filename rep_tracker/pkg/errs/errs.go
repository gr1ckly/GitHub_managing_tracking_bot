package errs

import "errors"

var (
	ErrInternal     = errors.New("internal error")
	ErrRepoNotFound = errors.New("not found")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidToken = errors.New("invalid token")
	ErrNotValidData = errors.New("not valid data")
)
