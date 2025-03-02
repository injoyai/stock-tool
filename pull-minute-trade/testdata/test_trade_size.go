package main

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"pull-minute-trade/model"
	"xorm.io/xorm"
)

func main() {
	c, err := tdx.DialHosts(tdx.Hosts)
	logs.PanicErr(err)

	code := "sz000001"

	resp, err := c.GetKlineDayAll(code)
	logs.PanicErr(err)

	b, err := sqlite.NewXorm("./" + code + ".db")
	logs.PanicErr(err)
	err = b.Sync2(new(model.Trade))
	logs.PanicErr(err)

	err = b.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			logs.Debug(v.Time)
			resp, err := c.GetHistoryMinuteTradeAll(v.Time.Format("20060102"), code)
			if err != nil {
				return err
			}
			logs.Debug(resp.Count)
			for _, vv := range resp.List {
				if _, err := session.Insert(&model.Trade{
					Time:   vv.Time,
					Price:  vv.Price.Int64(),
					Volume: vv.Volume,
					Status: vv.Status,
				}); err != nil {
					return err
				}
			}

		}
		return nil
	})
	logs.PanicErr(err)

}
