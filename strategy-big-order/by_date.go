package main

import (
	"fmt"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

func NewByCode(code string, start, end time.Time) *ByCode {
	return &ByCode{
		Code:  code,
		Start: start,
		End:   end,
	}
}

type ByCode struct {
	Code  string
	Start time.Time
	End   time.Time
}

func (this *ByCode) Run(m *tdx.Manage) error {
	pss := []Prices(nil)
	for i := this.Start; i.Before(this.End.Add(1)); i = i.Add(time.Hour * 24) {
		date := i.Format("20060102")
		if !m.Workday.Is(i) {
			continue
		}
		var ls protocol.Trades
		err := m.Do(func(c *tdx.Client) error {
			if date == time.Now().Format("20060102") {
				resp, err := c.GetTradeAll(this.Code)
				if err == nil {
					ls = resp.List
				}
				return err
			}
			resp, err := c.GetHistoryTradeAll(date, this.Code)
			if err == nil {
				ls = resp.List
			}
			return err
		})
		if err != nil {
			return err
		}
		ps := NewPrices(this.Code, i, ls)
		pss = append(pss, ps)
	}
	fmt.Println(m.Codes.GetName(this.Code))
	for _, v := range pss {
		fmt.Println(v)
	}
	return nil
}
