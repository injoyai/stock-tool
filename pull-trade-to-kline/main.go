package main

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"os"
	"path/filepath"
	"time"
)

var (
	Codes = []string{
		//"sz000001",
	}
	Indexes = []string{
		"sh000001",
		"sz399001",
		"sz399006",
	}
	Dir = "./data"
)

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: 3})
	logs.PanicErr(err)

	cs, err := tdx.GetBjCodes()
	logs.PanicErr(err)

	for _, v := range cs {
		Codes = append(Codes, "bj"+v.Code)
	}

	//PullIndexes(m, Indexes, Dir)

	PullStocks(m, Codes, Dir)

	//select {}
}

func PullStocks(m *tdx.Manage, codes []string, dir string) {
	dir = filepath.Join(dir, "股票")
	os.MkdirAll(dir, os.ModePerm)
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	if len(codes) == 0 {
		return
	}
	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("["+codes[0]+"]"),
		bar.WithFlush(),
	)
	defer b.Close()
	wg := chans.NewWaitLimit(10)
	for i, _ := range codes {
		code := codes[i]
		wg.Add()
		m.Go(func(c *tdx.Client) {
			b.SetPrefix("[" + code + "]")
			b.Flush()
			defer func() {
				b.Add(1)
				b.Flush()
				wg.Done()
			}()
			ls, err := pullTrades(c, m.Workday, code)
			if err != nil {
				logs.Err(err)
				return
			}
			ks1 := ls.Klines()
			save(ks1, dir, code, "1分钟")
			save(ks1.Merge(5), dir, code, "5分钟")
			save(ks1.Merge(15), dir, code, "15分钟")
			save(ks1.Merge(30), dir, code, "30分钟")
			save(ks1.Merge(60), dir, code, "60分钟")
		})
	}
	wg.Wait()
}

// PullIndexes 拉取指数
func PullIndexes(m *tdx.Manage, indexes []string, dir string) {
	if len(indexes) == 0 {
		return
	}
	dir = filepath.Join(dir, "指数")
	os.MkdirAll(dir, os.ModePerm)
	b := bar.New(
		bar.WithTotal(int64(len(indexes))),
		bar.WithPrefix("["+indexes[0]+"]"),
		bar.WithFlush(),
	)
	defer b.Close()
	wg := chans.NewWaitLimit(10)
	for i, _ := range indexes {
		code := indexes[i]
		wg.Add()
		m.Go(func(c *tdx.Client) {
			b.SetPrefix("[" + code + "]")
			b.Flush()
			defer func() {
				b.Add(1)
				b.Flush()
				wg.Done()
			}()
			ls, err := pullTrades(c, m.Workday, code)
			if err != nil {
				logs.Err(err)
				return
			}
			ks1 := ls.Klines()
			for ii, v := range ks1 {
				ks1[ii].Amount = protocol.Price(v.Volume * 100 * 1000)
				ks1[ii].Volume = 0
			}
			save(ks1, dir, code, "1分钟")
			save(ks1.Merge(5), dir, code, "5分钟")
			save(ks1.Merge(15), dir, code, "15分钟")
			save(ks1.Merge(30), dir, code, "30分钟")
			save(ks1.Merge(60), dir, code, "60分钟")
		})
	}
	wg.Wait()
}

func save(ks protocol.Klines, dir, code, typeName string) error {
	data := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
	}
	for _, v := range ks {
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
	filename := filepath.Join(dir, typeName, code+"-"+typeName+".csv")
	return oss.New(filename, buf)
}

func pullTrades(c *tdx.Client, w *tdx.Workday, code string) (protocol.Trades, error) {
	ls := protocol.Trades(nil)
	resp, err := c.GetKlineMonthAll(code)
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, nil
	}
	start := time.Date(resp.List[0].Time.Year(), resp.List[0].Time.Month(), 1, 0, 0, 0, 0, resp.List[0].Time.Location())
	var res *protocol.TradeResp
	w.Range(start, time.Now(), func(t time.Time) bool {
		res, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		ls = append(ls, res.List...)
		return true
	})

	return ls, nil
}
