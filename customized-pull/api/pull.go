package api

import (
	"context"
	"errors"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

func (this *Client) Pull(ctx context.Context, offset uint16, log func(s string), plan func(cu, to int), dealErr func(code string, err error)) error {

	//offset := uint16(g.InputVar("请输入偏移量:").Int())

	//c, err := tdx.DialDefault()
	//if err != nil {
	//	return err
	//}
	//
	//cs, err := tdx.NewCodes(c, "./codes.db")
	//if err != nil {
	//	return err
	//}

	//codes := cs.GetStocks()
	codes, err := this.GetCodes()
	if err != nil {
		return err
	}
	//logs.Debug(codes)
	//codes = []string{
	//	"sz000001",
	//}

	now := time.Now().Add(-time.Hour * 24 * time.Duration(offset))
	lastDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	ks6 := [6][]*protocol.Kline{}

	total := len(codes)
	plan(0, total)
	for i := range codes {
		code := codes[i]

		select {
		case <-ctx.Done():
			return errors.New("手动停止")
		default:
		}

		this.Pool.Go(func(c *tdx.Client) {

			defer func() {
				plan(i+1, total)
			}()

			resp, err := c.GetKlineMinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(lastDate)
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[0] = resp.List

			resp, err = c.GetKline5MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(lastDate.Add(-time.Hour * 24 * 2))
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[1] = resp.List

			resp, err = c.GetKline15MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(lastDate.Add(-time.Hour * 24 * 2))
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[2] = resp.List

			resp, err = c.GetKline30MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(lastDate.Add(-time.Hour * 24 * 3))
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[3] = resp.List

			resp, err = c.GetKlineHourUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(lastDate.Add(-time.Hour * 24 * 4))
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[4] = resp.List

			resp, err = c.GetKlineDay(code, offset, 30)
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[5] = resp.List

			err = klineToCsv(ks6, "./data/"+code+".csv", lastDate)
			if err != nil {
				dealErr(code, err)
				return
			}
		})
	}

	return nil
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
