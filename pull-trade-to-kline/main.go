package main

import (
	"path/filepath"
	"time"

	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

var (
	DatabaseDir   = "./data/database/kline"
	CsvDir        = "./data/csv"
	Clients       = 3
	Goroutine     = 6
	Startup       = true
	Retry         = 3
	RetryInterval = time.Second
	Indexes       = []string{
		//"sh999999", //"sh000001", //上证指数
		//"sz399001", //深证成指
		//"sz399006", //创业板指
		//"sh000016", //上证50
		//"sh000688", //科创50
		//"sh000010", //上证180
		//"sh000300", //沪深300
		//"sh000905", //中证500
		//"sh000852", //中证1000
		//"sh000932", //中证消费指数,
		//"sh000827", //中证环保指数,
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
	Start  = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	End    = time.Now().AddDate(0, -4, 0)
	CsvEnd = time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
)

func main() {
	m, err := tdx.NewManage(tdx.WithClients(Clients))
	logs.PanicErr(err)

	Indexes = m.Codes.GetIndexCodes()

	if len(Codes) == 0 {
		//Codes = m.Codes.GetStocks()
	}
	Codes = append(Codes, Indexes...)

	logs.PrintErr(pull(m, Codes))
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
	resp, err := c.GetKlineMonthAll(code)
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, nil
	}
	start := time.Date(resp.List[0].Time.Year(), resp.List[0].Time.Month(), 1, 0, 0, 0, 0, resp.List[0].Time.Location())
	var res *protocol.TradeResp
	w.Range(start, End, func(t time.Time) bool {
		res, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		ts = append(ts, res.List...)
		return true
	})
	return
}

func save(ks protocol.Klines, code string) error {
	//按年分割
	m := map[int]protocol.Klines{}
	for i := range ks {
		if indexesMap[code] {
			ks[i].Amount = protocol.Price(ks[i].Volume * 100 * 1000)
			ks[i].Volume = 0
		}
		m[ks[i].Time.Year()] = append(m[ks[i].Time.Year()], ks[i])
	}
	for year, ls := range m {

		k1 := toModel(ls)
		k5 := toModel(ls.Merge(5))
		k15 := toModel(ls.Merge(15))
		k30 := toModel(ls.Merge(30))
		k60 := toModel(ls.Merge(60))

		err := insertDB(year, code, k1, k5, k15, k30, k60)
		if err != nil {
			return err
		}
	}
	return exportCsv(ks, code)
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

func insertDB(year int, code string, k1, k5, k15, k30, k60 []any) error {
	if len(k1) == 0 {
		return nil
	}
	filename := filepath.Join(DatabaseDir, conv.String(year), code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Sync2(new(KlineMinute1), new(KlineMinute5), new(KlineMinute15), new(KlineMinute30), new(KlineMinute60)); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute1), k1); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute5), k5); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute15), k15); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute30), k30); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute60), k60); err != nil {
		return err
	}
	return nil
}

func _insertDB(db *xorms.Engine, table Timer, inserts []any) error {
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err := session.Where("ID>0").Delete(table); err != nil {
			return err
		}
		_, err := session.Table(table).Insert(inserts...)
		return err
	})
}

func exportCsv(ks protocol.Klines, code string) error {
	err := _exportCsv(ks, code, "1分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(5), code, "5分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(15), code, "15分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(30), code, "30分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(60), code, "60分钟")
	if err != nil {
		return err
	}
	return nil
}

func _exportCsv(ks protocol.Klines, code, _type string) error {
	data := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
	}
	for _, v := range ks {
		if v.Time.After(CsvEnd) {
			continue
		}
		data = append(data, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(CsvDir, _type, code+".csv")
	return oss.New(filename, buf)
}
