package domain

import "time"

type Session struct {
	user      *User
	token     string
	expiresAt time.Time
}

func NewSession(user *User, token string, expiresAt time.Time) Session {
	return Session{
		user:      user,
		token:     token,
		expiresAt: expiresAt,
	}
}

func (s Session) User() *User          { return s.user }
func (s Session) Token() string        { return s.token }
func (s Session) ExpiresAt() time.Time { return s.expiresAt }

func (s Session) Expired(now time.Time) bool {
	return !s.expiresAt.After(now)
}

func (s Session) WithExpiresAt(expiresAt time.Time) Session {
	s.expiresAt = expiresAt
	return s
}
