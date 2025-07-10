package main

import (
	"context"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

func NewExportKline(codes []string, coroutines int, years []int, database, export string) *ExportKline {
	return &ExportKline{
		Database:   database,
		Export:     export,
		Coroutines: coroutines,
		Codes:      codes,
		Years:      years,
	}
}

type ExportKline struct {
	Database   string
	Export     string
	Coroutines int
	Codes      []string
	Years      []int
}

func (this *ExportKline) Run(ctx context.Context, m *tdx.Manage) error {

	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	for _, year := range this.Years {
		logs.Debugf("导出年份: %d\n", year)

		b := bar.New()
		b.SetTotal(int64(len(codes)))
		b.SetFormat(func(b bar.Bar) string {
			return fmt.Sprintf("\r[导出] %s  %s  %s",
				b.Plan(),
				b.RateSize(),
				b.Speed(),
			)
		})

		limit := chans.NewWaitLimit(this.Coroutines)
		for i := range codes {
			code := codes[i]
			limit.Add()
			go func(code string) {
				defer limit.Done()
				defer func() {
					b.Add(1)
					b.Flush()
				}()

				filename := filepath.Join(this.Database, code+".db")
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
				err = this.export(db, code, year, new(KlineMinute1), "1分钟")
				logs.PrintErr(err)
				err = this.export(db, code, year, new(KlineMinute5), "5分钟")
				logs.PrintErr(err)
				err = this.export(db, code, year, new(KlineMinute15), "15分钟")
				logs.PrintErr(err)
				err = this.export(db, code, year, new(KlineMinute30), "30分钟")
				logs.PrintErr(err)
				err = this.export(db, code, year, new(KlineMinute60), "60分钟")
				logs.PrintErr(err)
			}(code)
		}
		limit.Wait()

		//进行压缩
		logs.Debug("进行压缩...")
		err := zip.Encode(
			filepath.Join(this.Export, conv.String(year)),
			filepath.Join(this.Export, conv.String(year)+".zip"),
		)
		logs.PrintErr(err)
	}

	return nil
}

func (this *ExportKline) export(db *xorms.Engine, code string, year int, table any, typeName string) error {
	data := []*KlineBase(nil)
	err := db.Table(table).Where("Year=?", year).Find(&data)
	if err != nil {
		return err
	}
	//logs.Debug(typeName, len(data))
	xx := [][]any{{
		"日期",
		"时间",
		"开盘",
		"最高",
		"最低",
		"收盘",
		"成交量",
		"成交额",
	}}
	for _, v := range data {
		xx = append(xx, []any{
			v.Time().Format(time.DateOnly),
			v.Time().Format("15:04"),
			v.Open,
			v.High,
			v.Low,
			v.Close,
			v.Volume,
			v.Amount,
		})
	}
	buf, err := csv.Export(xx)
	if err != nil {
		return err
	}
	return oss.New(filepath.Join(this.Export, conv.String(year), typeName, code+".csv"), buf)
}
