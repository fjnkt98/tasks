package domain

import "slices"

type User struct {
	id     int
	name   string
	digest string
	roles  []Role
}

func NewUser(id int, name, digest string) User {
	return User{
		id:     id,
		name:   name,
		digest: digest,
		roles:  nil,
	}
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

func (u User) WithDigest(digest string) User {
	u.digest = digest
	return u
}

func (u User) WithRoles(roles []Role) User {
	u.roles = slices.Clone(roles)
	return u
}

func (u User) WithRole(role Role) User {
	if u.HasRole(role) {
		return u
	}
	u.roles = append(slices.Clone(u.roles), role)
	return u
}

func (u User) WithoutRole(role Role) User {
	u.roles = slices.DeleteFunc(slices.Clone(u.roles), func(r Role) bool {
		return r == role
	})
	return u
}

func (u User) IsNew() bool {
	return u.id == 0
}
