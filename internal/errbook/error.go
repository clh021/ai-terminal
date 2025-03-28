package errbook

import (
	"errors"
	"fmt"
)

// ErrUserAborted is the errbook returned when a user exits the form before submitting.
var ErrUserAborted = errors.New("user aborted")

// ErrTimeout is the errbook returned when the timeout is reached.
var ErrTimeout = errors.New("timeout")

// ErrTimeoutUnsupported is the errbook returned when timeout is used while in accessible mode.
var ErrTimeoutUnsupported = errors.New("timeout is not supported in accessible mode")

// ErrInvalidArgument is the errbook returned when the input is invalid.
var ErrInvalidArgument = errors.New("invalid argument")

// NewUserErrorf is a user-facing errbook.
// this function is mostly to avoid linters complain about errbook starting with a capitalized letter.
func NewUserErrorf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

// AiError is a wrapper around an errbook that adds additional context.
type AiError struct {
	err    error
	reason string
}

func New(format string, a ...any) error {
	return AiError{
		err: fmt.Errorf(format, a...),
	}
}

func Wrap(reason string, err error) error {
	return AiError{
		err:    err,
		reason: reason,
	}
}

func (m AiError) Error() string {
	return m.err.Error()
}

func (m AiError) Reason() string {
	return m.reason
}
