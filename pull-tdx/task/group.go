package task

import (
	"context"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"time"
)

func Run(ctx context.Context, m *tdx.Manage, ls ...Tasker) error {
	return Group("", ls...).Run(ctx, m)
}

func Group(name string, tasks ...Tasker) *group {
	return &group{name: name, tasks: tasks}
}

type group struct {
	name  string
	tasks []Tasker
}

func (this *group) Name() string {
	return this.name
}

func (this *group) Run(ctx context.Context, m *tdx.Manage) error {
	for _, task := range this.tasks {
		start := time.Now()
		logs.Infof("任务[%s]开始...\n", task.Name())
		err := task.Run(ctx, m)
		logs.Infof("任务[%s]完成, 耗时: %s, 结果: %v\n", task.Name(), time.Since(start), conv.New(err).String("成功"))
		if err != nil {
			return err
		}
		fmt.Println("================================================================")
	}
	return nil
}
