package domain

const MaxPasswordLength = 72

type Password struct {
	value string
}

func NewPassword(value string) (Password, error) {
	if value == "" {
		return Password{}, ErrEmptyPassword
	}
	if len(value) > MaxPasswordLength {
		return Password{}, ErrPasswordTooLong
	}
	return Password{value: value}, nil
}

func (p Password) String() string {
	return p.value
}
