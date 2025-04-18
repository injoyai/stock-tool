package task

import (
	"context"
	"github.com/injoyai/tdx"
)

func NewGroup(name string, tasks ...Tasker) *Group {
	return &Group{name: name, tasks: tasks}
}

type Group struct {
	name  string
	tasks []Tasker
}

func (this *Group) Name() string {
	return this.name + "分组"
}

func (this *Group) Run(ctx context.Context, m *tdx.Manage) error {
	for _, v := range this.tasks {
		if err := Run(ctx, m, v); err != nil {
			return err
		}
	}
	return nil
}
