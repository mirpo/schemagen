package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaError_Chaining(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &SchemaError{
		Path:    "schema.json",
		Message: "invalid type",
		Cause:   cause,
	}

	assert.Equal(t, "schema schema.json: invalid type: underlying error", err.Error())
	assert.ErrorIs(t, err, cause)
}

func TestGenerationError_Chaining(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &GenerationError{
		Language: "go",
		File:     "types.go",
		Message:  "failed",
		Cause:    cause,
	}

	assert.ErrorIs(t, err, cause)
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

func TestSchemaError_NoCause(t *testing.T) {
	err := &SchemaError{
		Path:    "schema.json",
		Message: "missing field",
	}

	assert.Equal(t, "schema schema.json: missing field", err.Error())
	assert.NoError(t, err.Unwrap())
}

func TestGenerationError_NoCause(t *testing.T) {
	err := &GenerationError{
		Language: "python",
		File:     "models.py",
		Message:  "invalid type",
	}

	assert.Equal(t, "generating python models.py: invalid type", err.Error())
	assert.NoError(t, err.Unwrap())
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "output_dir",
		Message: "must be a valid path",
	}

	assert.Equal(t, "validation error for output_dir: must be a valid path", err.Error())
	assert.NoError(t, err.Unwrap())
}

func TestExitCodeError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := &ExitCodeError{
			Message: "command failed",
			Code:    ExitGeneral,
		}

		assert.Equal(t, "command failed", err.Error())
		assert.Equal(t, ExitGeneral, err.Code)
		assert.NoError(t, err.Unwrap())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := &ExitCodeError{
			Message: "command failed",
			Code:    ExitGeneral,
			Cause:   cause,
		}

		assert.Equal(t, "command failed: underlying error", err.Error())
		assert.ErrorIs(t, err, cause)
	})
}

func TestWrap(t *testing.T) {
	t.Run("wraps error", func(t *testing.T) {
		cause := fmt.Errorf("original error")
		err := Wrap(cause, "operation failed")

		assert.Equal(t, "operation failed: original error", err.Error())
		assert.Equal(t, ExitGeneral, err.Code)
		assert.ErrorIs(t, err, cause)
	})

	t.Run("nil error returns nil", func(t *testing.T) {
		err := Wrap(nil, "operation failed")
		assert.Nil(t, err)
	})
}
