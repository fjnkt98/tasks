package domain

import (
	"slices"
	"strings"
)

type User struct {
	id     int
	name   string
	digest string
	roles  []Role
}

func NewUser(id int, name, digest string, roles ...Role) (User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return User{}, ErrEmptyUserName
	}
	if digest == "" {
		return User{}, ErrEmptyDigest
	}
	return User{
		id:     id,
		name:   name,
		digest: digest,
		roles:  slices.Clone(roles),
	}, nil
}

func (u User) ID() int        { return u.id }
func (u User) Name() string   { return u.name }
func (u User) Digest() string { return u.digest }

func (u User) Roles() []Role {
	return slices.Clone(u.roles)
}

func (u User) HasRole(role Role) bool {
	return slices.Contains(u.roles, role)
}

func (u User) WithDigest(digest string) (User, error) {
	if digest == "" {
		return User{}, ErrEmptyDigest
	}
	u.digest = digest
	return u, nil
}

func (u User) GrantRole(role Role) User {
	if u.HasRole(role) {
		return u
	}
	u.roles = append(slices.Clone(u.roles), role)
	return u
}

func (u User) RevokeRole(role Role) User {
	u.roles = slices.DeleteFunc(slices.Clone(u.roles), func(r Role) bool {
		return r == role
	})
	return u
}

func (u User) IsNew() bool {
	return u.id == 0
}
