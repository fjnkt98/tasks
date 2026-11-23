package domain

import "time"

type Activity struct {
	userID    int
	kind      string
	message   string
	createdAt time.Time
}

func NewActivity(userID int, kind, message string, createdAt time.Time) Activity {
	return Activity{
		userID:    userID,
		kind:      kind,
		message:   message,
		createdAt: createdAt,
	}
}

func (a Activity) UserID() int          { return a.userID }
func (a Activity) Kind() string         { return a.kind }
func (a Activity) Message() string      { return a.message }
func (a Activity) CreatedAt() time.Time { return a.createdAt }
