package errors

import "fmt"

// SchemaError indicates a problem with schema loading or parsing
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

// GenerationError indicates a problem during code generation
type GenerationError struct {
	Language string
	File     string
	Message  string
	Cause    error
}

func (e *GenerationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("generating %s %s: %s: %v", e.Language, e.File, e.Message, e.Cause)
	}
	return fmt.Sprintf("generating %s %s: %s", e.Language, e.File, e.Message)
}

func (e *GenerationError) Unwrap() error {
	return e.Cause
}

// ValidationError indicates a config or input validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}
