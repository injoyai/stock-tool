package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

func main() {
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)

	codes := m.Codes.GetStocks()

	for i, _ := range codes {
		go func(code string) {
			err := m.Pool.Do(func(c *tdx.Client) error {
				ls := protocol.Trades(nil)
				resp, err := c.GetKlineMonthAll(code)
				if err != nil {
					return err
				}
				if len(resp.List) == 0 {
					return nil
				}
				start := time.Date(resp.List[0].Time.Year(), resp.List[0].Time.Month(), 1, 0, 0, 0, 0, resp.List[0].Time.Location())
				var res *protocol.TradeResp
				m.Workday.Range(start, time.Now(), func(t time.Time) bool {
					res, err = c.GetHistoryTradeAll(t.Format("20060102"), code)
					return err == nil
				})

				if err != nil {
					return err
				}
				ls = append(ls, res.List...)
				return nil
			})
			logs.PrintErr(err)
		}(codes[i])
	}

}
