package store

import "github.com/sipeed/picoclaw/pkg/reef"

// TaskStore is the persistence interface for Reef tasks.
type TaskStore interface {
	SaveTask(task *reef.Task) error
	GetTask(id string) (*reef.Task, error)
	UpdateTask(task *reef.Task) error
	DeleteTask(id string) error
	ListTasks(filter TaskFilter) ([]*reef.Task, error)
	SaveAttempt(taskID string, attempt reef.AttemptRecord) error
	GetAttempts(taskID string) ([]reef.AttemptRecord, error)
	Close() error
}
