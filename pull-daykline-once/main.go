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

func main() {

	c, err := tdx.DialDefault()
	logs.PanicErr(err)

	codes := tdx.DefaultCodes.GetStocks()
	//	codes = []string{"sz000001"}

	for _, code := range codes {
		resp, err := c.GetKlineDayUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local))
		})
		if err != nil {
			logs.Err(err)
			continue
		}

		data := [][]any{
			{"时间", "代码", "名称", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "涨幅", "涨幅比"},
		}

		if len(resp.List) > 1 {
			resp.List = resp.List[1 : len(resp.List)-1]
		}

		for _, v := range resp.List {
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
