package task

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/tdx/protocol"
	"pull-tdx/model"
	"time"
)

var (
	title = []any{"日期", "时间", "代码", "名称", "开盘", "最高", "最低", "收盘", "总手", "金额", "涨幅", "涨幅比"}
)

func klineToCsv2(code string, ks model.Klines, filename string, getName func(code string) string) error {
	ls := [][]any{title}
	for _, v := range ks {
		t := time.Unix(v.Date, 0)
		ls = append(ls, []any{
			t.Format(time.DateOnly),
			t.Format("15:04"),
			code,
			getName(code),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
		})
	}
	buf, err := excel.ToCsv(ls)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}

func klineToCsv(code string, ks []*protocol.Kline, filename string, getName func(code string) string) error {
	ls := [][]any{title}
	for _, v := range ks {
		ls = append(ls, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
			code,
			getName(code),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
		})
	}
	buf, err := excel.ToCsv(ls)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}
