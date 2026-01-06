package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

func Export(gb tdx.IGbbq, codes []string, goroutines int, year string, databaseDir, exportDir, uploadDir string) error {

	logs.Debugf("导出年份: %s\n", year)

	b := bar.NewCoroutine(len(codes), goroutines, bar.WithPrefix("[导出]"))
	defer b.Close()

	for i := range codes {
		code := codes[i]

		b.Go(func() {
			filename := filepath.Join(databaseDir, code, code+"-"+year+".db")
			if !oss.Exists(filename) {
				logs.Errf("文件不存在: %s\n", filename)
				return
			}
			db, err := sqlite.NewXorm(filename)
			if err != nil {
				logs.Err(err)
				return
			}
			defer db.Close()
			ks := protocol.Klines{}
			err = db.Find(&ks)
			if err != nil {
				logs.Err(err)
				return
			}

			err = exportYear(gb, ks, exportDir, year, "1分钟", code)
			logs.PrintErr(err)
			err = exportYear(gb, ks.Merge241(5), exportDir, year, "5分钟", code)
			logs.PrintErr(err)
			err = exportYear(gb, ks.Merge241(15), exportDir, year, "15分钟", code)
			logs.PrintErr(err)
			err = exportYear(gb, ks.Merge241(30), exportDir, year, "30分钟", code)
			logs.PrintErr(err)
			err = exportYear(gb, ks.Merge241(60), exportDir, year, "60分钟", code)
			logs.PrintErr(err)

		})

	}
	b.Wait()

	//进行压缩
	logs.Debug("压缩...")
	err := zip.Encode(
		filepath.Join(exportDir, year),
		filepath.Join(exportDir, year+".zip"),
	)
	logs.PrintErr(err)

	logs.Debug("重命名...")
	os.Rename(
		filepath.Join(exportDir, year+".zip"),
		filepath.Join(uploadDir, year+".zip"),
	)
	logs.PrintErr(err)

	return nil
}

func exportYear(gb tdx.IGbbq, ks protocol.Klines, dir, year, typeName, code string) error {
	xx := [][]any{Title}
	for _, v := range ks {
		x := []any{
			code,
			v.Time.Format(time.DateTime),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume * 100,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
			gb.GetTurnover(code, v.Time, v.Volume*100),
		}

		if eq := gb.GetEquity(code, v.Time); eq != nil {
			x = append(x, int64(eq.Float), int64(eq.Total))
		}
		xx = append(xx, x)
	}
	buf, err := csv.Export(xx)
	if err != nil {
		return err
	}
	return oss.New(filepath.Join(dir, year, typeName, code+".csv"), buf)
}
