package main

import (
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

var (
	Codes = []string{
		"sh600000",
		"sz000001",
	}
)

func main() {

	defer func() { g.Input("按回车键退出...") }()

	m, err := tdx.NewManage(nil, tdx.WithRedial())
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStocks()
	}

	err = doHistory(m, Codes, time.Date(1990, 1, 1, 0, 0, 0, 0, time.Local), time.Now())
	logs.PanicErr(err)
}

func doHistory(m *tdx.Manage, codes []string, start, end time.Time) error {

	all := make(map[string][]*Kline)

	for _, code := range codes {
		logs.Debug("开始执行:", code)
		err := m.Do(func(c *tdx.Client) error {
			resp, err := c.GetKlineDayAll(code)
			if err != nil {
				return err
			}
			for _, k := range resp.List {
				all[k.Time.Format(time.DateOnly)] = append(all[k.Time.Format(time.DateOnly)], &Kline{
					Code:  code,
					Name:  m.Codes.GetName(code),
					Kline: k,
				})
			}
			return nil
		})
		logs.PrintErr(err)
	}

	for i := start; i.Before(end); i = i.AddDate(0, 0, 1) {
		if !m.Workday.Is(i) {
			logs.Info(i.Format("2006-01-02") + "不是工作日")
			continue
		}

		data := [][]any{
			{"序号", "代码", "名称", "日期", "昨收", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "振幅", "涨跌幅"},
		}

		ls := all[i.Format(time.DateOnly)]

		for ii, l := range ls {
			data = append(data, []any{
				ii + 1,
				l.Code,
				l.Name,
				i.Format("2006-01-02"),
				l.Last.Float64(),
				l.Open.Float64(),
				l.Close.Float64(),
				l.High.Float64(),
				l.Low.Float64(),
				l.Volume,
				l.Amount.Float64(),
				l.RisePrice().Float64(),
				l.RiseRate(),
			})
		}
		buf, err := excel.ToCsv(data)
		if err != nil {
			logs.Err(err)
			continue
		}
		oss.New("./data/增量/日线/"+i.Format("2006/2006-01-02")+".csv", buf)

	}
	return nil

}

type Kline struct {
	Code string
	Name string
	*protocol.Kline
}
