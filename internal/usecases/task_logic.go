package usecases

import (
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

func PrepareAddTask(user domain.User, title string) (domain.Task, error) {
	return domain.NewTask(0, &user, title, domain.TaskStatusCreated)
}

func PrepareUpdateTaskTitle(task domain.Task, title string) (domain.Task, error) {
	return task.WithTitle(title)
}

func PrepareUpdateTaskStatus(task domain.Task, status string) (domain.Task, domain.TaskStatus, error) {
	st, err := domain.ParseTaskStatus(status)
	if err != nil {
		return domain.Task{}, "", err
	}
	task, err = task.WithStatus(st)
	if err != nil {
		return domain.Task{}, "", err
	}
	return task, st, nil
}

func EventTaskAdded(user *domain.User, task domain.Task) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindTask, fmt.Sprintf("add task %d, title: %q", task.ID(), task.Title()), time.Time{})
}

func EventTaskTitleUpdated(user *domain.User, task domain.Task) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindTask, fmt.Sprintf("edit task %d, title: %q", task.ID(), task.Title()), time.Time{})
}

func EventTaskStatusUpdated(user *domain.User, task domain.Task, status domain.TaskStatus) (Event, error) {
	action := "done"
	if status == domain.TaskStatusCreated {
		action = "undone"
	}
	return domain.NewActivity(user, domain.ActivityKindTask, fmt.Sprintf("%s task %d, %q", action, task.ID(), task.Title()), time.Time{})
}

func EventTaskDeleted(user *domain.User, taskID int) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindTask, fmt.Sprintf("delete task %d", taskID), time.Time{})
}
