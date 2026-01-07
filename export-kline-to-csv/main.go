package main

import (
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/injoyai/tdx/protocol"
)

const (
	DatabaseDir = "./data/database/kline241"
	ExportDir   = "./data/export/csv"
	Coroutines  = 10
)

func main() {

	gb, err := tdx.NewGbbq()
	logs.PanicErr(err)

	es, err := os.ReadDir(DatabaseDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutines, bar.WithPrefix("[xx000000]"))
	defer b.Close()

	for _, v := range es {
		dir := filepath.Join(DatabaseDir, v.Name())
		b.SetPrefix("[" + v.Name() + "]")
		b.GoRetry(func() error { return export(gb, dir, ExportDir) }, tdx.DefaultRetry)
	}

	b.Wait()

	logs.Info("完成...")

}

func export(gb *tdx.Gbbq, databaseDir, exportDir string) error {

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

	data := [][]any{
		extend.DefaultMinuteKlineExportTitle,
	}

	for _, v := range kss {
		x := []any{
			code, v.Time.Format("2006-01-02 15:04:05"),
			v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(),
			v.Volume * 100, v.Amount.Float64(),
			v.RisePrice().Float64(), v.RiseRate(),
			gb.GetTurnover(code, v.Time, v.Volume*100),
		}
		if eq := gb.GetEquity(code, v.Time); eq != nil {
			x = append(x, int64(eq.Float), int64(eq.Total))
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
