package main

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"xorm.io/xorm"
)

func main() {
	c, err := tdx.DialHosts(tdx.Hosts)
	logs.PanicErr(err)

	code := "sz000001"

	resp, err := c.GetKlineDayAll(code)
	logs.PanicErr(err)

	db, err := sqlite.NewXorm("./" + code + ".db")
	logs.PanicErr(err)
	err = db.Sync2(new(MinuteTrade))
	logs.PanicErr(err)

	err = db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			logs.Debug(v.Time)
			resp, err := c.GetHistoryMinuteTradeAll(v.Time.Format("20060102"), code)
			if err != nil {
				return err
			}
			logs.Debug(resp.Count)
			for _, vv := range resp.List {
				if _, err := session.Insert(&MinuteTrade{
					Time:   vv.Time,
					Price:  vv.Price.Float64(),
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

type MinuteTrade struct {
	Date   string
	Time   string
	Price  float64
	Volume int
	Status int
}
