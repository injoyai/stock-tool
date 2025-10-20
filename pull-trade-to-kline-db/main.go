package main

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/g"
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
	Dir           = "./data/database/kline"
	Clients       = 1
	Goroutine     = 2
	Startup       = true
	Retry         = 3
	RetryInterval = time.Second
	Indexes       = []string{
		"sh000001",
		"sz399001",
		"sz399006",
	}
	indexesMap = func() map[string]bool {
		m := make(map[string]bool)
		for _, v := range Indexes {
			m[v] = true
		}
		return m
	}()
	Codes = []string{
		//"sh600000",
	}
	Start = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	End   = time.Now().AddDate(0, -4, 0)
)

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	Codes = append(Codes, Indexes...)

	Run(m, Codes)
}

func Run(m *tdx.Manage, codes []string) {
	c := cron.New(cron.WithSeconds())
	c.AddFunc("0 10 15 * * *", func() {
		logs.PrintErr(pull(m, codes))
	})
	if Startup {
		logs.PrintErr(pull(m, codes))
	}
	c.Run()
}

func pull(m *tdx.Manage, codes []string) error {
	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("xx000000"),
		bar.WithFlush(),
	)
	defer b.Close()
	wg := chans.NewWaitLimit(Goroutine)
	for i, _ := range codes {
		wg.Add()
		go func(code string) {
			defer func() {
				b.Add(1)
				b.Flush()
				wg.Done()
			}()
			b.SetPrefix("[" + code + "]")
			b.Flush()
			var (
				ts  protocol.Trades
				err error
			)
			err = g.Retry(func() error {
				return m.Do(func(c *tdx.Client) error {
					ts, err = pullOne(c, m.Workday, code)
					return err
				})
			}, Retry, RetryInterval)
			if err != nil {
				b.Logf("[错误] [%s] %s", code, err)
				b.Flush()
				return
			}
			err = save(ts.Klines(), code)
			if err != nil {
				b.Logf("[错误] [%s] %s", code, err)
				b.Flush()
				return
			}
		}(codes[i])
	}
	wg.Wait()
	return nil
}

func pullOne(c *tdx.Client, w *tdx.Workday, code string) (ts protocol.Trades, err error) {
	var resp *protocol.TradeResp
	w.Range(Start, End, func(t time.Time) bool {
		resp, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		ts = append(ts, resp.List...)
		return true
	})
	return

	return c.GetHistoryTradeFull(code)
}

func save(ks protocol.Klines, code string) error {
	//按年分割
	m := map[int]protocol.Klines{}
	for _, v := range ks {
		if indexesMap[code] {
			v.Amount = protocol.Price(v.Volume * 100 * 1000)
			v.Volume = 0
		}
		m[v.Time.Year()] = append(m[v.Time.Year()], v)
	}
	for year, ls := range m {

		k1 := toModel(ls)
		k5 := toModel(ls.Merge(5))
		k15 := toModel(ls.Merge(15))
		k30 := toModel(ls.Merge(30))
		k60 := toModel(ls.Merge(60))

		err := insert(year, code, k1, k5, k15, k30, k60)
		if err != nil {
			return err
		}
	}
	return nil
}

func toModel(ks protocol.Klines) []any {
	inserts := make([]any, len(ks))
	for i, v := range ks {
		inserts[i] = &KlineBase{
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
			Volume: int(v.Volume),
			Amount: v.Amount.Float64(),
		}
	}
	return inserts
}

func insert(year int, code string, k1, k5, k15, k30, k60 []any) error {
	if len(k1) == 0 {
		return nil
	}
	filename := filepath.Join(Dir, conv.String(year), code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Sync2(new(KlineMinute1), new(KlineMinute5), new(KlineMinute15), new(KlineMinute30), new(KlineMinute60)); err != nil {
		return err
	}
	if err = _insert(db, new(KlineMinute1), k1); err != nil {
		return err
	}
	if err = _insert(db, new(KlineMinute5), k5); err != nil {
		return err
	}
	if err = _insert(db, new(KlineMinute15), k15); err != nil {
		return err
	}
	if err = _insert(db, new(KlineMinute30), k30); err != nil {
		return err
	}
	if err = _insert(db, new(KlineMinute60), k60); err != nil {
		return err
	}
	return nil
}

func _insert(db *xorms.Engine, table Timer, inserts []any) error {
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err := session.Where("ID>0").Delete(table); err != nil {
			return err
		}
		_, err := session.Table(table).Insert(inserts...)
		return err
	})
}
