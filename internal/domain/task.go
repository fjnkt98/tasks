package domain

import "fmt"

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
	userID int
	title  string
	status TaskStatus
}

func NewTask(id, userID int, title string, status TaskStatus) Task {
	return Task{
		id:     id,
		userID: userID,
		title:  title,
		status: status,
	}
}

func (t Task) ID() int            { return t.id }
func (t Task) UserID() int        { return t.userID }
func (t Task) Title() string      { return t.title }
func (t Task) Status() TaskStatus { return t.status }

func (t Task) WithTitle(title string) Task {
	t.title = title
	return t
}

func (t Task) WithStatus(status TaskStatus) Task {
	t.status = status
	return t
}
