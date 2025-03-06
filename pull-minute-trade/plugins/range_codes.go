package plugins

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/tdx"
)

type Range struct {
	Codes  []string
	Append []string
	limit  int
	m      *tdx.Manage
}

func (this *Range) Run(ctx context.Context, f func(code string)) error {

	//1. 获取所有股票代码
	codes := this.Codes
	if len(codes) == 0 {
		codes = this.m.Codes.GetStocks()
	}
	codes = append(codes, this.Append...)

	limit := chans.NewWaitLimit(uint(this.limit))

	for _, code := range codes {

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			limit.Add()
			go func(code string) {
				defer limit.Done()
				f(code)
			}(code)

		}

	}

	limit.Wait()

	return nil
}
