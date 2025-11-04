package task

import (
	"context"
	"github.com/injoyai/bar"
	"github.com/injoyai/tdx"
	"time"
)

type Range[T any] struct {
	Codes   []T        //股票代码
	Append  []T        //附加代码
	Limit   int        //并发数量
	Retry   int        //重试次数
	Handler Handler[T] //处理函数
}

func (this *Range[T]) Run(ctx context.Context, m *tdx.Manage) error {

	//1. 获取所有股票代码
	codes := this.Codes
	codes = append(codes, this.Append...)

	if this.Limit <= 0 {
		this.Limit = 1
	}

	b := bar.NewCoroutine(
		len(codes), this.Limit,
		bar.WithFormat(
			bar.WithText(time.Now().Format(time.TimeOnly)),
			bar.WithPlan(),
			bar.WithRateSize(),
			bar.WithSpeed(),
			bar.WithRemain(),
		),
		bar.WithFlush(),
	)

	for i := range codes {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:

			code := codes[i]
			b.GoRetry(func() error {
				err := this.Handler.Handler(ctx, m, code)
				if err != nil {
					b.Logf("[%s] 处理: %s, 失败: %v\n", this.Handler.Name(), code, err)
					b.Flush()
				}
				return err
			}, this.Retry)

		}
	}

	b.Wait()
	return nil
}
