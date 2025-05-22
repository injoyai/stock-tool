package task

import (
	"context"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/str/bar"
	"github.com/injoyai/logs"
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
	//if len(codes) == 0 {
	//	codes = m.Codes.GetStocks()
	//}
	codes = append(codes, this.Append...)

	if this.Limit <= 0 {
		this.Limit = 1
	}
	limit := chans.NewWaitLimit(uint(this.Limit))

	total := int64(len(codes))
	taskName := this.Handler.Name()
	logs.Tracef("[%s] 处理数量: %d\n", taskName, total)
	b := bar.New(total)
	b.AddOption(func(f *bar.Format) {
		f.Entity.SetFormatter(func(e *bar.Format) string {
			return fmt.Sprintf("\r%s [%s] %s  %s  %s  %-10s",
				time.Now().Format(time.TimeOnly),
				taskName,
				e.Bar,
				e.RateSize,
				e.Speed,
				e.Used,
			)
		})
	})
	b.Add(0).Flush()
	for _, code := range codes {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			limit.Add()
			go func(code T) {
				defer limit.Done()
				defer func() {
					b.Add(1).Flush()
				}()
				err := g.Retry(func() error { return this.Handler.Handler(ctx, m, code) }, this.Retry)
				if err != nil {
					logs.Errf("[%s] 处理: %s, 失败: %v\n", taskName, code, err)
				}
			}(code)

		}
	}

	limit.Wait()
	return nil
}
