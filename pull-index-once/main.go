package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"strings"
	"time"
)

var (
	Codes = []string{
		"sh999999", //上证指数
		"sz399001", //深证成指
	}
)

func main() {

	c, err := tdx.DialDefault()
	logs.PanicErr(err)
	c.Wait.SetTimeout(time.Second * 5)

	err = do(c.GetKlineDayAll, "日线_{code}")
	logs.PrintErr(err)

	err = do(c.GetKlineWeekAll, "周线_{code}")
	logs.PrintErr(err)

}

func do(f func(code string) (*protocol.KlineResp, error), name string) error {
	for _, code := range Codes {
		logs.Debug("开始拉取:", code)
		resp, err := f(code)
		logs.PanicErr(err)
		data := [][]any{
			{"日期", "开盘", "最高", "最低", "收盘", "成交量", "成交额", "涨跌价", "涨跌幅"},
		}
		for _, v := range resp.List {
			data = append(data, []any{v.Time.Format("2006-01-02"), v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(), v.Volume, v.Amount.Float64(), v.RisePrice().Float64(), v.RiseRate()})
		}
		buf, err := excel.ToCsv(data)
		logs.PanicErr(err)
		oss.New("./data/"+strings.ReplaceAll(name, "{code}", code)+".csv", buf)
	}
	return nil
}
