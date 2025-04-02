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

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.SetShowColor(false)
}

func main() {

	defer func() {
		if e := recover(); e != nil {
			logs.Err(e)
		}
		g.Input("按回车键结束...")
	}()

	offset := uint16(g.InputVar("请输入偏移量:").Int())

	c, err := tdx.DialDefault()
	logs.PanicErr(err)

	cs, err := tdx.NewCodes(c, "./codes.db")
	logs.PanicErr(err)

	codes := cs.GetStocks()
	//codes := []string{
	//	"sz000001",
	//}

	now := time.Now().Add(-time.Hour * 24 * time.Duration(offset))
	lastDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	ks6 := [6][]*protocol.Kline{}
	for _, code := range codes {
		logs.Debug(code)

		resp, err := c.GetKlineMinuteUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastDate)
		})
		logs.PanicErr(err)
		ks6[0] = resp.List

		resp, err = c.GetKline5MinuteUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastDate.Add(-time.Hour * 24 * 2))
		})
		logs.PanicErr(err)
		ks6[1] = resp.List

		resp, err = c.GetKline15MinuteUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastDate.Add(-time.Hour * 24 * 2))
		})
		logs.PanicErr(err)
		ks6[2] = resp.List

		resp, err = c.GetKline30MinuteUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastDate.Add(-time.Hour * 24 * 3))
		})
		logs.PanicErr(err)
		ks6[3] = resp.List

		resp, err = c.GetKlineHourUntil(code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastDate.Add(-time.Hour * 24 * 4))
		})
		logs.PanicErr(err)
		ks6[4] = resp.List

		resp, err = c.GetKlineDay(code, offset, 30)
		logs.PanicErr(err)
		ks6[5] = resp.List

		err = klineToCsv(ks6, "./data/"+code+".csv", lastDate)
		logs.PrintErr(err)
	}

}

var (
	title = []any{"日期", "时间", "总手", "金额"}
)

func klineToCsv(ks6 [6][]*protocol.Kline, filename string, lastDate time.Time) (err error) {
	lss := [][]any{
		{
			"日期", "时间", "总手", "金额", "", "", "",
			"日期", "时间", "总手", "金额", "", "", "",
			"日期", "时间", "总手", "金额", "", "", "",
			"日期", "时间", "总手", "金额", "", "", "",
			"日期", "时间", "总手", "金额", "", "", "",
			"日期", "时间", "总手", "金额", "", "", "",
		},
	}

	for i := 1; i <= 240; i++ {
		ls := []any(nil)
		for y := 0; y < 6; y++ {
			if len(ks6[y]) > i {
				v := ks6[y][i]
				if v.Time.Before(lastDate.Add(time.Hour * 24)) {
					ls = append(ls, []any{
						v.Time.Format(time.DateTime),
						v.Time.Format("15:04"),
						v.Volume,
						v.Amount.Float64(),
						"", "", "",
					}...)
					continue
				}

			}
			ls = append(ls, []any{
				"",
				"",
				"",
				"",
				"", "", "",
			}...)
		}
		lss = append(lss, ls)
	}

	buf, err := excel.ToCsv(lss)
	if err != nil {
		logs.Err(err)
		return err
	}
	return oss.New(filename, buf)
}
