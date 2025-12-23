package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/lib/xorms"
	"github.com/injoyai/tdx/protocol"
)

var (
	TradeDir   = "./data/database/trade"
	KlineDir   = "./data/database/kline"
	Coroutines = 10
	Retry      = tdx.DefaultRetry
)

func main() {

	es, err := os.ReadDir(TradeDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutines, bar.WithPrefix("[shxxxxxx]"))
	defer b.Close()

	for _, v := range es {
		func() {
			defer b.Done()
			b.SetPrefix("[" + v.Name() + "]")
			b.Flush()
			dir := filepath.Join(TradeDir, v.Name())
			err = oss.RangeFile(dir, func(info *oss.FileInfo, f *os.File) (bool, error) {
				filename := filepath.Join(dir, f.Name())
				exportDir := filepath.Join(KlineDir, v.Name())
				err = g.Retry(func() error { return export(filename, exportDir) }, Retry)
				logs.PrintErr(err)
				return true, nil
			})
			b.Log("[错误]", err)
			b.Flush()
		}()
	}

	b.Wait()
	logs.Info("完成...")
}

func export(filename string, exportDir string) error {
	if !oss.Exists(filename) {
		return nil
	}
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	data := []*Trade(nil)
	err = db.Find(&data)
	if err != nil {
		return err
	}

	ts := make(protocol.Trades, 0, len(data))
	for _, v := range data {
		ts = append(ts, v.To())
	}

	xx := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量(手)", "成交额", "涨跌", "涨跌幅(%)", "流通股本(股)", "总股本(股)", "换手率(%)"},
	}
	for _, v := range ts.Klines() {
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
			0,
			0,
			0,
		})
	}

	return nil
}
