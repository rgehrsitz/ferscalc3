package api

import "fmt"

// ErrorKind classifies high-level error categories for UI and API callers.
type ErrorKind string

const (
	ErrorKindValidation  ErrorKind = "validation"
	ErrorKindConfig      ErrorKind = "config"
	ErrorKindCalculation ErrorKind = "calculation"
)

// AppError wraps errors with a category and optional field metadata.
type AppError struct {
	Kind    ErrorKind
	Field   string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Field != "" {
		return fmt.Sprintf("%s error (%s): %s", e.Kind, e.Field, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Kind, e.Message)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
