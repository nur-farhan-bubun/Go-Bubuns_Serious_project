package errors

import "fmt"

// AppError represents a structured application error.
type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NotFound creates a 404 error.
func NotFound(entity string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: entity + " not found",
		Status:  404,
	}
}

// Internal creates a 500 error.
func Internal(msg string, err error) *AppError {
	return &AppError{
		Code:    "INTERNAL",
		Message: msg,
		Status:  500,
		Err:     err,
	}
}

// BadRequest creates a 400 error.
func BadRequest(msg string) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: msg,
		Status:  400,
	}
}

// Unauthorized creates a 401 error.
func Unauthorized(msg string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: msg,
		Status:  401,
	}
}

// Forbidden creates a 403 error.
func Forbidden(msg string) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: msg,
		Status:  403,
	}
}
