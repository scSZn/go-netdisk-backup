package group

import "context"

type TaskInterface interface {
	Run(ctx context.Context) error
}

type Task struct {
	Name     string
	RunField func(ctx context.Context) error
}

func (t *Task) Run(ctx context.Context) error {
	return t.RunField(ctx)
}
