package main

import (
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
)

var (
	End       = time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	Dir       = cfg.GetString("dir", "./data/trade")
	Clients   = cfg.GetInt("clients", 3)
	Coroutine = cfg.GetInt("coroutine", 10)
	Codes     = cfg.GetStrings("codes")
	After     = cfg.GetString("after")
)

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStocks()
	}

	b := bar.NewCoroutine(len(Codes), Coroutine)
	defer b.Close()

	for i := range Codes {
		code := Codes[i]
		if code < After {
			b.Add(1)
			b.Flush()
			continue
		}
		b.Go(func() {
			b.SetPrefix("[" + code + "]")
			b.Flush()
			var resp protocol.Trades
			err = m.Do(func(c *tdx.Client) error {
				resp, err = c.GetHistoryTradeBefore(code, m.Workday, End)
				return err
			})
			if err != nil {
				b.Logf("[ERR] [%s] %v", code, err)
				b.Flush()
				return
			}
			err = save(resp, code)
			if err != nil {
				b.Logf("[ERR] [%s] %v", code, err)
				b.Flush()
				return
			}
		})
	}

	b.Wait()

}

func save(ts protocol.Trades, code string) error {
	data := [][]any{
		{"时间", "价格", "成交量", "方向(0买,1卖,2中性)"},
	}
	for _, v := range ts {
		data = append(data, []any{
			v.Time.Format(time.DateTime),
			v.Price.Float64(),
			v.Volume,
			v.Status,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(Dir, code+".csv")
	return oss.New(filename, buf)
}
