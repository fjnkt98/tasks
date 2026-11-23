package domain

import (
	"errors"
	"fmt"
)

var ErrValidation = errors.New("validation error")

var (
	ErrInvalidTaskStatus    = fmt.Errorf("%w: invalid task status", ErrValidation)
	ErrEmptyTaskTitle       = fmt.Errorf("%w: task title is required", ErrValidation)
	ErrTaskUserRequired     = fmt.Errorf("%w: task user is required", ErrValidation)
	ErrEmptyUserName        = fmt.Errorf("%w: username is required", ErrValidation)
	ErrEmptyDigest          = fmt.Errorf("%w: digest is required", ErrValidation)
	ErrEmptyPassword        = fmt.Errorf("%w: password is required", ErrValidation)
	ErrPasswordTooLong      = fmt.Errorf("%w: password too long", ErrValidation)
	ErrSessionUserRequired  = fmt.Errorf("%w: session user is required", ErrValidation)
	ErrEmptySessionToken    = fmt.Errorf("%w: session token is required", ErrValidation)
	ErrActivityUserRequired = fmt.Errorf("%w: activity user is required", ErrValidation)
	ErrEmptyActivityKind    = fmt.Errorf("%w: activity kind is required", ErrValidation)
	ErrEmptyActivityMessage = fmt.Errorf("%w: activity message is required", ErrValidation)
)
