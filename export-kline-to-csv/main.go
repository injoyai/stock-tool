package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

const (
	DatabaseDir = "./data/database/kline241"
	ExportDir   = "./data/export/csv"
	Coroutines  = 10
	After       = ""
)

var (
	Table = []any{"日期", "开盘", "最高", "最低", "收盘", "成交量(股)", "成交额(元)", "涨跌(元)", "涨跌幅(%)", "换手率(%)", "流通股本(股)", "总股本(股)"}
)

func main() {

	m, err := tdx.NewManage(
		tdx.WithWorkday(&tdx.Workday{}),
		tdx.WithCodes(tdx.NewCodesBase()),
		tdx.WithDialGbbqDefault(),
	)
	logs.PanicErr(err)

	es, err := os.ReadDir(DatabaseDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutines, bar.WithPrefix("[xx000000]"))
	defer b.Close()

	for _, v := range es {
		code := v.Name()
		dir := filepath.Join(DatabaseDir, v.Name())
		b.SetPrefix("[" + v.Name() + "]")
		b.GoRetry(func() error {
			if code < After {
				return nil
			}
			return export(m, dir, ExportDir)
		}, tdx.DefaultRetry)
	}

	b.Wait()

	logs.Info("完成...")

}

func export(m *tdx.Manage, databaseDir, exportDir string) error {

	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
		}
	}()

	code := filepath.Base(databaseDir)
	kss := protocol.Klines{}
	err := oss.RangeFileInfo(databaseDir, func(info *oss.FileInfo) (bool, error) {
		ks, err := loading(info.FullName())
		if err != nil {
			return false, err
		}
		kss = append(kss, ks...)
		return true, nil
	})
	if err != nil {
		return err
	}
	kss.Sort()

	if len(kss) == 0 {
		return nil
	}

	//获取上市日期
	startDate := times.IntegerDay(kss[0].Time)
	err = m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineDayAll(code)
		if err != nil {
			return err
		}
		if len(resp.List) > 0 {
			startDate = times.IntegerDay(resp.List[0].Time)
		}
		return nil
	})
	logs.PrintErr(err)

	kss2 := protocol.Klines{}
	for _, v := range kss {
		if v.Time.After(startDate) {
			kss2 = append(kss2, v)
		}
	}

	err = toCsv(m.Gbbq, kss2, filepath.Join(exportDir, "1分钟"), code)
	if err != nil {
		return err
	}

	err = toCsv(m.Gbbq, kss2.Merge241(5), filepath.Join(exportDir, "5分钟"), code)
	if err != nil {
		return err
	}

	err = toCsv(m.Gbbq, kss2.Merge241(15), filepath.Join(exportDir, "15分钟"), code)
	if err != nil {
		return err
	}

	err = toCsv(m.Gbbq, kss2.Merge241(30), filepath.Join(exportDir, "30分钟"), code)
	if err != nil {
		return err
	}

	err = toCsv(m.Gbbq, kss2.Merge241(60), filepath.Join(exportDir, "60分钟"), code)
	if err != nil {
		return err
	}

	return nil
}

func loading(filename string) (protocol.Klines, error) {
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ks := protocol.Klines{}
	err = db.Find(&ks)
	return ks, err
}

func toCsv(gb tdx.IGbbq, kss protocol.Klines, exportDir, code string) error {
	kss.Sort()

	data := [][]any{Table}

	for i, v := range kss {
		//修复集合竞价没有的情况
		if v.Time.Format(time.TimeOnly) == "09:30:00" && v.Open == 0 && i+1 < len(kss) {
			if i > 0 {
				v.Last = kss[i-1].Close
			}
			v.Open = kss[i+1].Open
			v.High = kss[i+1].Open
			v.Low = kss[i+1].Open
			v.Close = kss[i+1].Open
		}
		x := []any{
			v.Time.Format("2006-01-02 15:04:05"),
			v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(),
			v.Volume * 100, v.Amount.Float64(),
			v.RisePrice().Float64(), v.RiseRate(),
			gb.GetTurnover(code, v.Time, v.Volume*100),
		}
		if eq := gb.GetEquity(code, v.Time); eq != nil {
			x = append(x, eq.Float, eq.Total)
		}
		data = append(data, x)
	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}

	filename := filepath.Join(exportDir, code+".csv")
	return oss.New(filename, buf)
}
