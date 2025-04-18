package task

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
)

type Range struct {
	Codes   []string
	Append  []string
	Limit   int
	Handler func(code string) error
}

func (this *Range) Run(ctx context.Context, m *tdx.Manage) error {

	//1. 获取所有股票代码
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	codes = append(codes, this.Append...)

	if this.Limit <= 0 {
		this.Limit = 1
	}
	limit := chans.NewWaitLimit(uint(this.Limit))

	logs.Trace("处理数量:", len(codes))
	for _, code := range codes {
		logs.Tracef("处理: %s\n", code)

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			limit.Add()
			go func(code string) {
				defer limit.Done()
				err := this.Handler(code)
				logs.PrintErr(err)
			}(code)
		}

	}

	limit.Wait()

	return nil
}
