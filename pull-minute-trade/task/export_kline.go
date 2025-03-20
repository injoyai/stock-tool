package task

import (
	"context"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
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

func (this *ExportKline) Run(ctx context.Context, m *tdx.Manage) error {

	for _, v := range this.Type {
		switch v {
		case "code":
			if err := this.byCode(ctx, m); err != nil {
				return err
			}
		case "date":
			if err := this.byDate(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *ExportKline) byDate(ctx context.Context, m *tdx.Manage) error {
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

func (this *ExportKline) byCode(ctx context.Context, m *tdx.Manage) error {
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
			{"序号", "代码", "名称", "日期", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "涨幅", "涨幅比"},
		}
		for _, v := range all {
			data = append(data, []any{
				v.Date, code, m.Codes.GetName(code), v.Open.Float64(), v.Close.Float64(), v.High.Float64(), v.Low.Float64(), v.Volume, v.Amount.Float64(), v.RisePrice(), v.RiseRate(),
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
