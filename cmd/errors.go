package cmd

import "errors"

const (
	exitGeneral = 1
	exitDrift   = 2
)

type exitCodeError struct {
	Message string
	Code    int
	Cause   error
}

func (e *exitCodeError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *exitCodeError) Unwrap() error {
	return e.Cause
}

func ExitCode(err error) int {
	var e *exitCodeError
	if errors.As(err, &e) {
		return e.Code
	}
	return 1
}

func wrapErr(err error, msg string) *exitCodeError {
	if err == nil {
		return nil
	}
	return &exitCodeError{
		Message: msg,
		Code:    exitGeneral,
		Cause:   err,
	}
}
