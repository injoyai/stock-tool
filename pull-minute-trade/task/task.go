package task

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
)

func Run(ctx context.Context, i Tasker) error {
	logs.Infof("任务[%s]开始...\n", i.Name())
	err := i.Run(ctx)
	logs.Infof("任务[%s]完成, 结果: %v\n", i.Name(), conv.New(err).String("成功"))
	return err
}
