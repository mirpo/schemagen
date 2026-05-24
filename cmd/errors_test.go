package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitCode(t *testing.T) {
	t.Run("exitCodeError returns code", func(t *testing.T) {
		err := &exitCodeError{Code: 2, Message: "drift detected"}
		assert.Equal(t, 2, ExitCode(err))
	})

	t.Run("generic error returns 1", func(t *testing.T) {
		assert.Equal(t, 1, ExitCode(fmt.Errorf("generic")))
	})

	t.Run("wrapped exitCodeError returns code", func(t *testing.T) {
		inner := &exitCodeError{Code: 2, Message: "drift"}
		wrapped := fmt.Errorf("outer: %w", inner)
		assert.Equal(t, 2, ExitCode(wrapped))
	})
}

func TestExitCodeError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		err := &exitCodeError{Message: "failed", Cause: fmt.Errorf("boom")}
		assert.Equal(t, "failed: boom", err.Error())
	})

	t.Run("without cause", func(t *testing.T) {
		err := &exitCodeError{Message: "failed"}
		assert.Equal(t, "failed", err.Error())
	})
}

func TestWrapErr(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.Nil(t, wrapErr(nil, "msg"))
	})

	t.Run("wraps error", func(t *testing.T) {
		cause := fmt.Errorf("root")
		wrapped := wrapErr(cause, "context")
		assert.Equal(t, exitGeneral, wrapped.Code)
		assert.Equal(t, "context", wrapped.Message)
		assert.Equal(t, cause, wrapped.Cause)
	})
}
