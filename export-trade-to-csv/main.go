package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/lib/xorms"
	"github.com/injoyai/tdx/protocol"
)

var (
	TradeDir   = "./data/database/trade"
	csvDir     = "./data/database/csv"
	Coroutines = 10
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
}

func main() {

	gb, err := tdx.NewGbbq()
	logs.PanicErr(err)

	es, err := os.ReadDir(TradeDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutines, bar.WithPrefix("[shxxxxxx]"))
	defer b.Close()

	for _, v := range es {
		b.Go(func() {
			b.SetPrefix("[" + v.Name() + "]")
			b.Flush()
			dir := filepath.Join(TradeDir, v.Name())
			var ts protocol.Trades
			err = oss.RangeFileInfo(dir, func(info *oss.FileInfo) (bool, error) {
				filename := filepath.Join(dir, info.Name())
				_ts, err := read(filename)
				if err != nil {
					return false, err
				}
				ts = append(ts, _ts...)
				return true, nil
			})
			if err != nil {
				b.Log("[错误]", err)
				b.Flush()
				return
			}
			err = export(ts.Klines(), csvDir, v.Name(), gb)
			if err != nil {
				b.Log("[错误]", err)
				b.Flush()
				return
			}
		})
	}

	b.Wait()
	logs.Info("done...")
}

func read(filename string) (protocol.Trades, error) {
	if !oss.Exists(filename) {
		return nil, nil
	}
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	data := []*Trade(nil)
	err = db.Find(&data)
	if err != nil {
		return nil, err
	}

	ts := make(protocol.Trades, 0, len(data))
	for _, v := range data {
		ts = append(ts, v.To())
	}
	return ts, nil
}

func export(ks protocol.Klines, exportDir, code string, gb *tdx.Gbbq) error {

	exportName := filepath.Join(exportDir, "1分钟", code+".csv")
	save(gb, code, exportName, ks)

	exportName = filepath.Join(exportDir, "5分钟", code+".csv")
	save(gb, code, exportName, ks.Merge241(5))

	exportName = filepath.Join(exportDir, "15分钟", code+".csv")
	save(gb, code, exportName, ks.Merge241(15))

	exportName = filepath.Join(exportDir, "30分钟", code+".csv")
	save(gb, code, exportName, ks.Merge241(30))

	exportName = filepath.Join(exportDir, "60分钟", code+".csv")
	save(gb, code, exportName, ks.Merge241(60))

	return nil
}

func save(gb *tdx.Gbbq, code, filename string, ks protocol.Klines) error {
	ks.Sort()
	xx := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量(手)", "成交额", "涨跌", "涨跌幅(%)", "流通股本(股)", "总股本(股)", "换手率(%)"},
	}
	for _, v := range ks {
		e := gb.GetEquity(code, v.Time)
		xx = append(xx, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
			e.Float,
			e.Total,
			e.Turnover(v.Volume * 100),
		})
	}

	buf, err := csv.Export(xx)
	if err != nil {
		return err
	}

	return oss.New(filename, buf)
}
