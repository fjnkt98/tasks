package domain

import "time"

type Activity struct {
	user      *User
	kind      string
	message   string
	createdAt time.Time
}

func NewActivity(user *User, kind, message string, createdAt time.Time) Activity {
	return Activity{
		user:      user,
		kind:      kind,
		message:   message,
		createdAt: createdAt,
	}
}

func (a Activity) User() *User          { return a.user }
func (a Activity) Kind() string         { return a.kind }
func (a Activity) Message() string      { return a.message }
func (a Activity) CreatedAt() time.Time { return a.createdAt }
