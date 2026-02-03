package errors

import "fmt"

// SchemaError indicates a problem with schema loading or parsing.
type SchemaError struct {
	Path    string
	Message string
	Cause   error
}

func (e *SchemaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("schema %s: %s: %v", e.Path, e.Message, e.Cause)
	}
	return fmt.Sprintf("schema %s: %s", e.Path, e.Message)
}

func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// GenerationError indicates a problem during code generation.
type GenerationError struct {
	Language string
	File     string
	Message  string
	Cause    error
}

func (e *GenerationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf(
			"generating %s %s: %s: %v",
			e.Language,
			e.File,
			e.Message,
			e.Cause,
		)
	}
	return fmt.Sprintf(
		"generating %s %s: %s",
		e.Language,
		e.File,
		e.Message,
	)
}

func (e *GenerationError) Unwrap() error {
	return e.Cause
}

// ValidationError indicates a config or input validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// Unwrap returns nil as ValidationError doesn't wrap other errors.
func (e *ValidationError) Unwrap() error {
	return nil
}

// ExitCodeError is an error that carries an exit code for CLI commands.
type ExitCodeError struct {
	Message string
	Code    int
	Cause   error
}

func (e *ExitCodeError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error for error chain support.
func (e *ExitCodeError) Unwrap() error {
	return e.Cause
}

// Common exit codes
const (
	ExitGeneral = 1 // General error
	ExitUsage   = 2 // Invalid usage/arguments
)

// NewUsageError creates an error for invalid CLI usage.
func NewUsageError(msg string) *ExitCodeError {
	return &ExitCodeError{Message: msg, Code: ExitUsage}
}

// Wrap wraps an error with context and returns an ExitCodeError.
// The original error is preserved in the error chain via Unwrap.
func Wrap(err error, msg string) *ExitCodeError {
	if err == nil {
		return nil
	}
	return &ExitCodeError{
		Message: msg,
		Code:    ExitGeneral,
		Cause:   err,
	}
}
