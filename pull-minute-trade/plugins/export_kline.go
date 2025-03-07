package plugins

import (
	"context"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"time"
)

type ExportKline struct {
	From string   //数据来源
	To   string   //保存位置
	Type []string //按代码,按日期
	*Range
}

func (this *ExportKline) Name() string {
	return "导出k线数据"
}

func (this *ExportKline) Run(ctx context.Context) error {
	return nil
}

func (this *ExportKline) byDate(ctx context.Context) error {
	now := time.Now().Format("20060102")
	ls := model.Klines{}
	err := this.Range.Run(ctx, func(code string) {
		//取每个代码的最新当日数据
		filename := filepath.Join(this.From, code+".db")
		err := db.WithOpen(filename, func(db *db.Sqlite) error {
			one := new(model.Kline)
			has, err := db.Where("Date=?", now).Get(one)
			if err != nil {
				return err
			} else if has {
				ls = append(ls, one)
			}
			return nil
		})
		logs.PrintErr(err)
	})
	if err != nil {
		return err
	}

	//
	ls.Sort()
	data := [][]any{
		{"序号", "代码", "名称"},
	}
	for i, v := range ls {
		data = append(data, []any{
			i + 1, v.Code, this.Range.m.Codes.GetName(v.Code),
		})
	}
	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}
	return oss.New(filepath.Join(this.To, now+".csv"), buf)
}

func (this *ExportKline) byCode(ctx context.Context) error {
	return this.Range.Run(ctx, func(code string) {
		filename := filepath.Join(this.From, code+".db")
		all := []*model.Kline(nil)
		err := db.WithOpen(filename, func(db *db.Sqlite) error {
			return db.Asc("Date").Find(&all)
		})
		if err != nil {
			logs.Err(err)
			return
		}

		//
		data := [][]any{
			{"序号", "日期", "开盘", "收盘", "最高", "最低", "成交量", "成交额"},
		}
		for _, v := range all {
			data = append(data, []any{
				v.Date, v.Open, v.Close, v.High, v.Low, v.Volume, v.Amount,
			})
		}
		buf, err := excel.ToCsv(data)
		if err != nil {
			logs.Err(err)
			return
		}
		oss.New(filepath.Join(this.To, code+".csv"), buf)
	})
}
