package main

import (
	"github.com/injoyai/base/types"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"time"
)

type ByCode struct {
	Codes []string
}

func (this *ByCode) Run(m *tdx.Manage) error {
	now := time.Now()
	p3s := types.List[Prices]{}
	for _, code := range Codes {
		err := m.Go(func(c *tdx.Client) {
			resp, err := c.GetTradeAll(code)
			if err != nil {
				logs.PrintErr(err)
				return
			}
			if len(resp.List) == 0 {
				return
			}
			p3 := NewPrices(code, now, resp.List)
			p3s = append(p3s, p3)
		})
		logs.PrintErr(err)
	}

	p3s.Sort(func(a, b Prices) bool {
		return a.SmallRate() < b.SmallRate()
	})

	for _, v := range p3s {
		logs.Debug(v.String())
	}
	return nil
}
