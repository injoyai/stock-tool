package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"time"
)

var (
	Codes = []string{
		//"sh999999", //上证指数
		//"sh000300", //沪深300
		//"sz399852",//中证1000
		//"sz399001", //深证成指
	}
)

func main() {

	c, err := tdx.DialDefault()
	logs.PanicErr(err)
	c.Wait.SetTimeout(time.Second * 5)

	for _, code := range Codes {
		logs.Debug("开始拉取:", code)
		resp, err := c.GetKlineDayAll(code)
		logs.PanicErr(err)

		data := [][]any{
			{"日期", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
		}

		for _, v := range resp.List {
			data = append(data, []any{v.Time.Format("2006-01-02"), v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(), v.Volume, v.Amount.Float64()})
		}

		buf, err := excel.ToCsv(data)
		logs.PanicErr(err)

		oss.New("./data/"+code+".csv", buf)

	}

}
