package task

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
)

type Tasker interface {
	Name() string
	Run(ctx context.Context) error
	//Running() bool
	//RunInfo() string
}

func Run(ctx context.Context, ls ...Tasker) error {
	for _, i := range ls {
		logs.Infof("任务[%s]开始...\n", i.Name())
		err := i.Run(ctx)
		logs.Infof("任务[%s]完成, 结果: %v\n", i.Name(), conv.New(err).String("成功"))
		if err != nil {
			return err
		}
	}
	return nil
}
