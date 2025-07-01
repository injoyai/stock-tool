package main

import (
	"fmt"
	"github.com/injoyai/base/types"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

var (
	Boundary = [2]protocol.Price{100 * 1e7, 10 * 1e7}
	Limit    = -1
)

func main() {

	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)

	p3s := types.List[Prices]{}
	for _, code := range m.Codes.GetStocks(Limit) {
		err = m.Go(func(c *tdx.Client) {
			resp, err := c.GetTradeAll(code)
			if err != nil {
				logs.PrintErr(err)
				return
			}
			if len(resp.List) == 0 {
				return
			}
			p3 := Prices{Code: code}
			for _, v := range resp.List {
				p := v.Amount()
				switch {
				case p >= Boundary[0]:
					if v.Status == 0 {
						p3.Big += p
					} else if v.Status == 1 {
						p3.Big -= p
					}
				case p >= Boundary[1]:
					if v.Status == 0 {
						p3.Mid += p
					} else if v.Status == 1 {
						p3.Mid -= p
					}
				default:
					if v.Status == 0 {
						p3.Small += p
					} else if v.Status == 1 {
						p3.Small -= p
					}
				}
			}
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
}

type Prices struct {
	Code            string
	Big, Mid, Small protocol.Price
}

func (p Prices) BigRate() float64 {
	return p.Big.Float64() / p.Sum().Float64()
}

func (p Prices) SmallRate() float64 {
	return p.Small.Float64() / p.Sum().Float64()
}

func (p Prices) Sum() protocol.Price {
	return p.Big + p.Mid + p.Small
}

func (p Prices) String() string {
	return fmt.Sprintf("%s: 大单: %.1f%%, 中单: %.1f%%, 小单: %.1f%%",
		p.Code,
		p.Big.Float64()/p.Sum().Float64()*100,
		p.Mid.Float64()/p.Sum().Float64()*100,
		p.Small.Float64()/p.Sum().Float64()*100,
	)
}
