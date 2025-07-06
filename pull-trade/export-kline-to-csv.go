package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

func NewExportKline(codes []string, years []int, database, export string) *ExportKline {
	return &ExportKline{
		Database: database,
		Export:   export,
		Codes:    codes,
		Years:    years,
	}
}

type ExportKline struct {
	Database string
	Export   string
	Codes    []string
	Years    []int
}

func (this *ExportKline) Run(ctx context.Context, m *tdx.Manage) error {

	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	for _, year := range this.Years {
		logs.Debugf("执行年份: %d\n", year)
		for _, code := range codes {
			err := func() error {
				filename := filepath.Join(this.Database, code+".db")
				if !oss.Exists(filename) {
					return fmt.Errorf("文件不存在: %s", filename)
				}
				db, err := sqlite.NewXorm(filename)
				if err != nil {
					return err
				}
				defer db.Close()
				this.export(db, code, year, new(KlineMinute1), "1分钟")
				this.export(db, code, year, new(KlineMinute5), "5分钟")
				this.export(db, code, year, new(KlineMinute15), "15分钟")
				this.export(db, code, year, new(KlineMinute30), "30分钟")
				this.export(db, code, year, new(KlineMinute60), "60分钟")
				return nil
			}()
			logs.PrintErr(err)
		}
	}

	return nil
}

func (this *ExportKline) export(db *xorms.Engine, code string, year int, table any, typeName string) error {
	data := []*KlineBase(nil)
	err := db.Table(table).Where("Year=?", year).Find(&data)
	if err != nil {
		return err
	}
	logs.Debug(typeName, len(data))
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
