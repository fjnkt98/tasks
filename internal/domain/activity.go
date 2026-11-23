package domain

import (
	"strings"
	"time"
)

type Activity struct {
	user      *User
	kind      string
	message   string
	createdAt time.Time
}

func NewActivity(user *User, kind, message string, createdAt time.Time) (Activity, error) {
	if user == nil {
		return Activity{}, ErrActivityUserRequired
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return Activity{}, ErrEmptyActivityKind
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return Activity{}, ErrEmptyActivityMessage
	}
	return Activity{
		user:      user,
		kind:      kind,
		message:   message,
		createdAt: createdAt,
	}, nil
}

func (a Activity) User() *User          { return a.user }
func (a Activity) Kind() string         { return a.kind }
func (a Activity) Message() string      { return a.message }
func (a Activity) CreatedAt() time.Time { return a.createdAt }
