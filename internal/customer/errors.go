package customer

import "errors"

var (
	// ErrNotFound indicates that the requested customer resource does not exist.
	ErrNotFound     = errors.New("not found")
	// ErrForbidden indicates the authenticated actor lacks permission for the operation.
	ErrForbidden    = errors.New("forbidden")
	// ErrValidation indicates input or state validation failed.
	ErrValidation   = errors.New("validation error")
	// ErrConflict indicates a state conflict, such as uniqueness or transition constraints.
	ErrConflict     = errors.New("conflict")
	// ErrUnauthorized indicates missing or invalid actor authentication context.
	ErrUnauthorized = errors.New("unauthorized")
)
