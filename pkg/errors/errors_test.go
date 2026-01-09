package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaError(t *testing.T) {
	// Basic error message
	err := &SchemaError{
		Path:    "schema.json",
		Message: "invalid type",
	}
	assert.Equal(t, "schema schema.json: invalid type", err.Error())

	// With cause
	cause := fmt.Errorf("underlying error")
	errWithCause := &SchemaError{
		Path:    "schema.json",
		Message: "invalid type",
		Cause:   cause,
	}

	assert.Equal(
		t,
		"schema schema.json: invalid type: underlying error",
		errWithCause.Error(),
	)
	assert.ErrorIs(t, errWithCause, cause)
}

func TestGenerationError(t *testing.T) {
	err := &GenerationError{
		Language: "typescript",
		File:     "types.ts",
		Message:  "failed",
	}
	assert.Equal(t, "generating typescript types.ts: failed", err.Error())

	// With cause
	cause := fmt.Errorf("underlying error")
	errWithCause := &GenerationError{
		Language: "go",
		File:     "types.go",
		Message:  "failed",
		Cause:    cause,
	}

	assert.ErrorIs(t, errWithCause, cause)
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "language",
		Message: "must not be empty",
	}

	assert.Equal(
		t,
		"validation error for language: must not be empty",
		err.Error(),
	)

	// ValidationError should not wrap anything
	assert.NotErrorIs(t, err, fmt.Errorf("anything"))
}

func TestErrorChaining(t *testing.T) {
	rootCause := fmt.Errorf("root cause")

	schemaErr := &SchemaError{
		Path:    "test.json",
		Message: "schema error",
		Cause:   rootCause,
	}

	genErr := &GenerationError{
		Language: "typescript",
		File:     "types.ts",
		Message:  "generation failed",
		Cause:    schemaErr,
	}

	require.ErrorIs(t, genErr, rootCause)

	var se *SchemaError
	require.ErrorAs(t, genErr, &se)
	assert.Equal(t, "test.json", se.Path)

	var ve *ValidationError
	assert.NotErrorAs(t, genErr, &ve)
}

func TestExitCodeError(t *testing.T) {
	err := &ExitCodeError{
		Message: "missing argument",
		Code:    2,
	}
	assert.Equal(t, "missing argument", err.Error())
	assert.Equal(t, 2, err.Code)
}

func TestNewUsageError(t *testing.T) {
	err := NewUsageError("invalid flag")
	assert.Equal(t, "invalid flag", err.Error())
	assert.Equal(t, ExitUsage, err.Code)
}
