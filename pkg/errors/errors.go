// Package errors defines domain-level sentinel errors and maps them to gRPC status codes.
package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = NewSentinel("not found")
	// ErrConflict is returned when a resource conflict occurs.
	ErrConflict = NewSentinel("conflict")
	// ErrAlreadyExists is returned when attempting to create a duplicate resource.
	ErrAlreadyExists = NewSentinel("already exists")
	// ErrValidation is returned when input validation fails.
	ErrValidation = NewSentinel("validation error")
	// ErrUnauthorized is returned when authentication is missing or invalid.
	ErrUnauthorized = NewSentinel("unauthorized")
	// ErrForbidden is returned when the caller lacks permission.
	ErrForbidden = NewSentinel("forbidden")
	// ErrIdempotencyKey is returned when an idempotency key conflict occurs.
	ErrIdempotencyKey = NewSentinel("idempotency key error")
	// ErrRateLimited is returned when the caller exceeds the rate limit.
	ErrRateLimited = NewSentinel("rate limited")
)

// Sentinel is a simple sentinel error type that supports equality checks via errors.Is.
type Sentinel struct {
	msg string
}

// NewSentinel creates a new sentinel error with the given message.
func NewSentinel(msg string) *Sentinel {
	return &Sentinel{msg: msg}
}

func (s *Sentinel) Error() string {
	return s.msg
}

// MapToGRPC converts a domain sentinel error into the appropriate gRPC status error.
func MapToGRPC(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case ErrNotFound:
		return status.Error(codes.NotFound, err.Error())
	case ErrConflict:
		return status.Error(codes.AlreadyExists, err.Error())
	case ErrAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	case ErrValidation:
		return status.Error(codes.InvalidArgument, err.Error())
	case ErrUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case ErrForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case ErrIdempotencyKey:
		return status.Error(codes.AlreadyExists, err.Error())
	case ErrRateLimited:
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Error(codes.Unknown, err.Error())
	}
}
