package pipeline

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerationError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		err := &GenerationError{Language: "go", File: "user.go", Message: "write failed", Cause: fmt.Errorf("disk full")}
		assert.Equal(t, "generating go user.go: write failed: disk full", err.Error())
	})

	t.Run("without cause", func(t *testing.T) {
		err := &GenerationError{Language: "ts", File: "user.ts", Message: "invalid type"}
		assert.Equal(t, "generating ts user.ts: invalid type", err.Error())
	})
}

func TestGenerationError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &GenerationError{Cause: cause}
	assert.Equal(t, cause, errors.Unwrap(err))

	err2 := &GenerationError{}
	assert.NoError(t, errors.Unwrap(err2))
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "name", Message: "required"}
	assert.Equal(t, "validation error for name: required", err.Error())
}

func TestValidationError_NoUnwrap(t *testing.T) {
	err := &ValidationError{Field: "x", Message: "bad"}
	assert.NoError(t, errors.Unwrap(err))
}
