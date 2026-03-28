// Package errors provides RFC 7807 Problem Details types for structured API
// error responses.
package errors

import "fmt"

// ProblemDetail represents an RFC 7807 error response.
type ProblemDetail struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	Errors   []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single field-level validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface so ProblemDetail can be used as a Go
// error value.
func (p ProblemDetail) Error() string {
	if p.Detail != "" {
		return p.Detail
	}
	return p.Title
}

// NewValidationError creates a ProblemDetail for request validation failures
// (HTTP 400).
func NewValidationError(detail string, errs ...ValidationError) ProblemDetail {
	return ProblemDetail{
		Type:   "https://problems.oms.io/validation-error",
		Title:  "Validation Error",
		Status: 400,
		Detail: detail,
		Errors: errs,
	}
}

// NewNotFoundError creates a ProblemDetail for missing resources (HTTP 404).
func NewNotFoundError(resource, id string) ProblemDetail {
	return ProblemDetail{
		Type:   "https://problems.oms.io/not-found",
		Title:  "Resource Not Found",
		Status: 404,
		Detail: fmt.Sprintf("%s with id %q not found", resource, id),
	}
}

// NewConflictError creates a ProblemDetail for state conflict errors (HTTP 409).
func NewConflictError(detail string) ProblemDetail {
	return ProblemDetail{
		Type:   "https://problems.oms.io/conflict",
		Title:  "Conflict",
		Status: 409,
		Detail: detail,
	}
}
