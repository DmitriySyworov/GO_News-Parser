package sherrors

import (
	"errors"
	"strings"
)

var (
	ErrInvalidAccessToken           = errors.New("invalid access token")
	ErrFailedAssertionContextValues = errors.New("failed to assert type ContextValues: ")
	ErrCriticalServer               = errors.New("critical error on the server side")
	ErrIncorrectTypeRemove          = errors.New("the type  must be a soft-delete or hard-delete")
	ErrIncorrectLimit               = errors.New("the limit must be a positive integer not greater than 100")
	ErrIncorrectOffset              = errors.New("the offset must be a positive integer")
	ErrRateLimiting                 = errors.New("the limit for sending requests per minute has been exceeded")
)

type MapError struct {
	Map map[string]string
}

func (m MapError) Error() string {
	sliceError := make([]string, 0, 5)
	for _, value := range m.Map {
		sliceError = append(sliceError, value)
	}
	return strings.Join(sliceError, ";")
}
