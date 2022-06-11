package exceptions

import (
	"errors"
)

var (
	ErrInvalidResponseWriter = errors.New("invalid response writer")
	ErrIncompleteEnvironment = errors.New("INCOMPLETE_ENVIRONMENT")
	ErrNotImplemented        = errors.New("NOT_IMPLEMENTED")
	ErrInvalidImageSize      = errors.New("INVALID_IMAGE_SIZE")
	ErrInvalidJobPriority    = errors.New("INVALID_JOB_PRIORITY")
	ErrUnauthorised          = errors.New("UNAUTHORISED")
	ErrInternalServerError   = errors.New("INTERNAL_SERVER_ERROR")
)
