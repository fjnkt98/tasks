package domain

type User struct {
	id     int
	name   string
	digest string
}

func NewUser(id int, name, digest string) User {
	return User{
		id:     id,
		name:   name,
		digest: digest,
	}
}

func (u User) ID() int        { return u.id }
func (u User) Name() string   { return u.name }
func (u User) Digest() string { return u.digest }

func (u User) WithDigest(digest string) User {
	u.digest = digest
	return u
}
