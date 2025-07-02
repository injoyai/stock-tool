package main

import (
	"fmt"
	"github.com/injoyai/tdx/protocol"
	"time"
)

func NewPrices(code string, date time.Time, ts protocol.Trades) Prices {
	p3 := Prices{Code: code, Date: date}
	for i, v := range ts {
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
		if i == len(ts)-1 {
			p3.Price = v.Price
		}
	}
	return p3
}

type Prices struct {
	Code            string
	Date            time.Time
	Big, Mid, Small protocol.Price
	Price           protocol.Price
}

func (p Prices) BigRate() float64 {
	if p.Sum() == 0 {
		return 0
	}
	return p.Big.Float64() / p.Sum().Float64()
}

func (p Prices) MidRate() float64 {
	if p.Sum() == 0 {
		return 0
	}
	return p.Small.Float64() / p.Sum().Float64()
}

func (p Prices) SmallRate() float64 {
	if p.Sum() == 0 {
		return 0
	}
	return p.Small.Float64() / p.Sum().Float64()
}

func (p Prices) Sum() protocol.Price {
	return p.Big + p.Mid + p.Small
}

func (p Prices) String() string {
	return fmt.Sprintf("%s %s: %.2f 大单: %.1f%%(%.1f万), 中单: %.1f%%(%.1f万), 小单: %.1f%%(%.1f万)",
		p.Date.Format("2006-01-02"),
		p.Code,
		p.Price.Float64(),
		p.BigRate()*100, p.Big.Float64()/10000,
		p.MidRate()*100, p.Mid.Float64()/10000,
		p.SmallRate()*100, p.Small.Float64()/10000,
	)
}
