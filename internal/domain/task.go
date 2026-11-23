package domain

import (
	"fmt"
	"strings"
)

type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusDone    TaskStatus = "done"
)

func ParseTaskStatus(s string) (TaskStatus, error) {
	switch TaskStatus(s) {
	case TaskStatusCreated, TaskStatusDone:
		return TaskStatus(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidTaskStatus, s)
	}
}

type Task struct {
	id     int
	user   *User
	title  string
	status TaskStatus
}

func NewTask(id int, user *User, title string, status TaskStatus) (Task, error) {
	if user == nil {
		return Task{}, ErrTaskUserRequired
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTaskTitle
	}
	if _, err := ParseTaskStatus(string(status)); err != nil {
		return Task{}, err
	}
	return Task{
		id:     id,
		user:   user,
		title:  title,
		status: status,
	}, nil
}

func (t Task) ID() int            { return t.id }
func (t Task) User() *User        { return t.user }
func (t Task) Title() string      { return t.title }
func (t Task) Status() TaskStatus { return t.status }

func (t Task) WithTitle(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTaskTitle
	}
	t.title = title
	return t, nil
}

func (t Task) WithStatus(status TaskStatus) (Task, error) {
	if _, err := ParseTaskStatus(string(status)); err != nil {
		return Task{}, err
	}
	t.status = status
	return t, nil
}

func (t Task) IsDone() bool {
	return t.status == TaskStatusDone
}

func (t Task) IsNew() bool {
	return t.id == 0
}
