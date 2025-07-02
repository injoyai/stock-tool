package main

import (
	"fmt"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

func NewByDate(code string, start, end time.Time) *ByDate {
	return &ByDate{
		Code:  code,
		Start: start,
		End:   end,
	}
}

type ByDate struct {
	Code  string
	Start time.Time
	End   time.Time
}

func (this *ByDate) Run(m *tdx.Manage) error {
	pss := []Prices(nil)
	for i := this.Start; i.Before(this.End.Add(1)); i = i.Add(time.Hour * 24) {
		date := i.Format("20060102")
		if !m.Workday.Is(i) {
			continue
		}
		var resp *protocol.HistoryTradeResp
		var err error
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetHistoryTradeAll(date, this.Code)
			return err
		})
		if err != nil {
			return err
		}
		ps := NewPrices(this.Code, i, resp.List)
		pss = append(pss, ps)
	}
	fmt.Println(m.Codes.GetName(this.Code))
	for _, v := range pss {
		fmt.Println(v)
	}
	return nil
}
