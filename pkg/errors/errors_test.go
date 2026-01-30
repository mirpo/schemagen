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
