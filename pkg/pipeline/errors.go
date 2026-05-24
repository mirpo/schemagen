package pipeline

import "fmt"

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

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}
