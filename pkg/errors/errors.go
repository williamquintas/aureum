package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound       = NewSentinel("not found")
	ErrConflict       = NewSentinel("conflict")
	ErrAlreadyExists  = NewSentinel("already exists")
	ErrValidation     = NewSentinel("validation error")
	ErrUnauthorized   = NewSentinel("unauthorized")
	ErrForbidden      = NewSentinel("forbidden")
	ErrIdempotencyKey = NewSentinel("idempotency key error")
	ErrRateLimited    = NewSentinel("rate limited")
)

type Sentinel struct {
	msg string
}

func NewSentinel(msg string) *Sentinel {
	return &Sentinel{msg: msg}
}

func (s *Sentinel) Error() string {
	return s.msg
}

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
