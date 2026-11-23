package domain

import "time"

type Session struct {
	userID    int
	token     string
	expiresAt time.Time
}

func NewSession(userID int, token string, expiresAt time.Time) Session {
	return Session{
		userID:    userID,
		token:     token,
		expiresAt: expiresAt,
	}
}

func (s Session) UserID() int          { return s.userID }
func (s Session) Token() string        { return s.token }
func (s Session) ExpiresAt() time.Time { return s.expiresAt }

func (s Session) Expired(now time.Time) bool {
	return !s.expiresAt.After(now)
}

func (s Session) WithExpiresAt(expiresAt time.Time) Session {
	s.expiresAt = expiresAt
	return s
}
