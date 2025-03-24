package main

import (
	"errors"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"time"
)

func main() {

	defer func() { g.Input("按回车键退出...") }()

	m, err := tdx.NewManage(nil, tdx.WithRedial())
	logs.PanicErr(err)

	err = do(m, m.Codes.GetStocks())
	logs.PrintErr(err)

	cr := cron.New(cron.WithSeconds())
	cr.AddFunc("0 10 15 * *", func() { do(m, m.Codes.GetStocks()) })
	cr.Run()
}

func do(m *tdx.Manage, codes []string) error {

	if !m.Workday.TodayIs() {
		return errors.New("今天不是工作日")
	}

	defer logs.Spend("耗时")()

	data := [][]any{
		{"序号", "代码", "名称", "日期", "昨收", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "振幅", "涨跌幅"},
	}

	for i, code := range codes {
		err := m.Do(func(c *tdx.Client) (err error) {
			resp, err := c.GetKlineDay(code, 0, 2)
			if err == nil && len(resp.List) == 2 {
				var l *protocol.Kline
				switch len(resp.List) {
				case 1:
					l = resp.List[0]
				case 2:
					l = resp.List[1]
				default:
					return
				}

				data = append(data, []any{
					i + 1,
					code,
					m.Codes.GetName(code),
					time.Now().Format("2006-01-02"),
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
			return err
		})
		logs.PrintErr(err)
	}

	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}
	return oss.New("./data/增量/日线/"+time.Now().Format("2006/2006-01-02")+".csv", buf)
}
