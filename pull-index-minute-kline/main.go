package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"time"
	"xorm.io/xorm"
)

var (
	Dir   = "./data/database/kline"
	Codes = []string{
		"sh000001",
		"sz399001",
		"sz399006",
	}
	Startup = true
)

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: 3})
	logs.PanicErr(err)

	c := cron.New(cron.WithSeconds())
	c.AddFunc("0 1 15 * * *", func() { Run(m) })

	if Startup {
		Run(m)
	}

	c.Run()
}

func Run(m *tdx.Manage) {

	b := bar.NewCoroutine(len(Codes), 3)
	defer b.Close()

	for i := range Codes {
		code := Codes[i]
		b.Go(func() {
			b.SetPrefix("[" + code + "]")
			b.Flush()
			err := m.Do(func(c *tdx.Client) error {
				return update(c, m.Workday, code)
			})
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err.Error())
				b.Flush()
			}
		})
	}

	b.Wait()

}

func update(c *tdx.Client, w *tdx.Workday, code string) error {
	dir := filepath.Join(Dir, conv.String(time.Now().Year()))
	filename := filepath.Join(dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	last := new(KlineMinute1)
	_, err = db.Desc("Date").Get(last)
	if err != nil {
		return err
	}

	if last.Date == 0 {
		last.Date = time.Now().AddDate(0, -4, 0).Unix()
	}

	ks := []*KlineBase(nil)
	w.Range(time.Unix(last.Date, 0).AddDate(0, 0, 1), time.Now(), func(t time.Time) bool {
		var resp *protocol.TradeResp
		resp, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		for _, v := range resp.List.Klines() {
			ks = append(ks, &KlineBase{
				Date:   v.Time.Unix(),
				Year:   v.Time.Year(),
				Month:  int(v.Time.Month()),
				Day:    v.Time.Day(),
				Hour:   v.Time.Hour(),
				Minute: v.Time.Minute(),
				Open:   v.Open.Float64(),
				High:   v.High.Float64(),
				Low:    v.Low.Float64(),
				Close:  v.Close.Float64(),
				Volume: 0,
				Amount: float64(v.Volume * 100),
			})
		}
		return true
	})
	if err != nil {
		return err
	}

	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range ks {
			_, err = session.Table(new(KlineMinute1)).Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})

}

func export() {

}
