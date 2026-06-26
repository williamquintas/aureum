package domain

import "errors"

var (
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrMissingField     = errors.New("required field is missing")
	ErrNoData           = errors.New("no data available for the requested period")
	ErrAccessDenied     = errors.New("access denied: record does not belong to user")
)
