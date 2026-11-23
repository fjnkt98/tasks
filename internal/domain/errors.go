package domain

import "errors"

var ErrNotFound = errors.New("not found")
var ErrUniqueConstraint = errors.New("unique constraint violation")
var ErrInvalidCredentials = errors.New("invalid credentials")
