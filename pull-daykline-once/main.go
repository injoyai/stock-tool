package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
)

var (
	codes = []string{"sh600797", "sh600519", "sh601899"}
	start = time.Date(2021, 10, 1, 0, 0, 0, 0, time.Local)
	end   = time.Date(2022, 9, 30, 23, 0, 0, 0, time.Local)
)

func main() {

	c, err := tdx.DialDefault()
	logs.PanicErr(err)

	if len(codes) == 0 {
		codes = tdx.DefaultCodes.GetStocks()
	}

	for _, code := range codes {
		resp, err := c.GetKlineDayUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(start)
		})
		if err != nil {
			logs.Err(err)
			continue
		}

		data := [][]any{
			{"时间", "代码", "名称", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "涨幅", "涨幅比"},
		}

		if len(resp.List) > 0 {
			resp.List = resp.List[1:]
		}

		for _, v := range resp.List {
			if v.Time.After(end) {
				break
			}
			data = append(data, []any{
				v.Time.Format("2006-01-02"),
				code,
				tdx.DefaultCodes.GetName(code),
				v.Open.Float64(),
				v.Close.Float64(),
				v.High.Float64(),
				v.Low.Float64(),
				v.Volume,
				v.Amount.Float64(),
				v.RisePrice().Float64(),
				float64(v.RisePrice()) / float64(v.Last) * 100,
			})
		}

		buf, err := excel.ToCsv(data)
		if err != nil {
			logs.Err(err)
			continue
		}

		oss.New(filepath.Join("./data/csv", code+".csv"), buf)

	}

}
