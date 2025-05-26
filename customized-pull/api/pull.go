package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"strings"
	"time"
)

func (this *Client) Pull(ctx context.Context, lastDate time.Time, log func(s string), plan func(cu, to int), dealErr func(code string, err error), day [6]int, avgDecimal, avg2Scale, avg2Decimal, limit int) error {

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

	dateList := []time.Time(nil)
	err := this.Pool.Do(func(c *tdx.Client) error {
		resp, err := c.GetIndexDay("sh000001", 0, 800)
		if err != nil {
			return err
		}
		for _, v := range resp.List {
			if v.Time.Before(lastDate.Add(time.Hour * 16)) {
				dateList = append(dateList, time.Date(v.Time.Year(), v.Time.Month(), v.Time.Day(), 0, 0, 0, 0, time.Local))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	//codes := cs.GetStocks()
	codes, err := this.GetCodes()
	if err != nil {
		return err
	}
	//logs.Debug(codes)
	//codes = []string{
	//	"sz000001",
	//}

	if len(codes) > limit {
		codes = codes[:limit]
	}

	//now := time.Now().Add(-time.Hour * 24 * time.Duration(offset))
	//lastDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, time.Local)
	lastDate = lastDate.Add(time.Hour * 23)

	total := len(codes)
	plan(0, total)
	for i := range codes {
		code := codes[i]
		ks6 := [6][]*protocol.Kline{}
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
				return k.Time.Before(dateList[len(dateList)-day[0]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[0] = resp.List

			resp, err = c.GetKline5MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(dateList[len(dateList)-day[1]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[1] = resp.List

			resp, err = c.GetKline15MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(dateList[len(dateList)-day[2]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[2] = resp.List

			resp, err = c.GetKline30MinuteUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(dateList[len(dateList)-day[3]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[3] = resp.List

			resp, err = c.GetKlineHourUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(dateList[len(dateList)-day[4]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[4] = resp.List

			resp, err = c.GetKlineDayUntil(code, func(k *protocol.Kline) bool {
				return k.Time.Before(dateList[len(dateList)-day[5]])
			})
			if err != nil {
				dealErr(code, err)
				return
			}
			ks6[5] = resp.List

			err = klineToCsv(ks6, filepath.Join(this.Dir, lastDate.Format("2006-01-02"), code+".csv"), lastDate, avgDecimal, avg2Scale, avg2Decimal)
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

func klineToCsv(ks6 [6][]*protocol.Kline, filename string, lastDate time.Time, avgDecimal, avg2Scale, avg2Decimal int) (err error) {
	lss := [][]any{
		{
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
			"日期", "时间", "总手", "5行均值", "10行均值", "金额", "5行均值", "10行均值", "",
		},
	}

	for i := 1; i <= 240; i++ {
		ls := []any(nil)
		for y := 0; y < 6; y++ {
			if len(ks6[y]) > i {
				v := ks6[y][i]
				if v.Time.Before(lastDate) {
					x := []any{
						v.Time.Format("2006/01/02"),
						v.Time.Format("1504"),
						v.Volume,
						func() string {
							if len(lss) >= 5 {
								total := float64(v.Volume)
								for _, vv := range lss[len(lss)-4:] {
									total += float64(vv[2+y*9].(int64))
								}
								return fmt.Sprintf(fmt.Sprintf("%%0.%df", avgDecimal), total/5)
								return conv.String(g.Decimals(total/5, avgDecimal))
							}
							return ""
						}(),
						func() string {
							if len(lss) >= 10 {
								total := float64(v.Volume)
								for _, vv := range lss[len(lss)-9:] {
									total += float64(vv[2+y*9].(int64))
								}
								return fmt.Sprintf(fmt.Sprintf("%%0.%df", avgDecimal), total/10)
								return conv.String(g.Decimals(total/10, avgDecimal))
							}
							return ""
						}(),
						v.Amount.Float64() / float64(avg2Scale),
						func() string {
							if len(lss) >= 5 {
								total := v.Amount.Float64() / float64(avg2Scale)
								for _, vv := range lss[len(lss)-4:] {
									total += conv.Float64(vv[5+y*9])
								}
								return fmt.Sprintf(fmt.Sprintf("%%0.%df", avg2Decimal), total/5)
								return conv.String(g.Decimals(total/5, avgDecimal))
							}
							return ""
						}(),
						func() string {
							if len(lss) >= 10 {
								total := v.Amount.Float64() / float64(avg2Scale)
								for _, vv := range lss[len(lss)-9:] {
									total += conv.Float64(vv[5+y*9])
								}
								return fmt.Sprintf(fmt.Sprintf("%%0.%df", avg2Decimal), total/10)
								return conv.String(g.Decimals(total/10, avgDecimal))
							}
							return ""
						}(),
						"",
					}
					ls = append(ls, x...)
					continue
				}

			}
			ls = append(ls, []any{
				"",
				"",
				"",
				"", "",
				"",
				"", "",
				"",
			}...)
		}
		lss = append(lss, ls)
	}

	for _, ls := range lss {
		for y := 0; y < 6; y++ {
			if f, ok := ls[5+y*9].(float64); ok {
				ls[5+y*9] = fmt.Sprintf(fmt.Sprintf("%%0.%df", avg2Decimal), f)
			}
		}
	}

	buf, err := excel.ToCsv(lss)
	if err != nil {
		logs.Err(err)
		return err
	}
	return oss.New(filename, buf)
}

func (this *Client) PullMinuteTrade(ctx context.Context, plan func(cu, to int), dealErr func(code string, err error), limit int) error {

	lastDate := time.Now()

	codes, err := this.GetCodes()
	if err != nil {
		return err
	}
	//logs.Debug(codes)
	//codes = []string{
	//	"sz000001",
	//}

	if len(codes) > limit {
		codes = codes[:limit]
	}

	total := len(codes)
	plan(0, total)
	lss := [][]any{{"代码", "日期", "925分"}}
	for i := 930; i <= 945; i++ {
		lss[0] = append(lss[0], fmt.Sprintf("%d分", i))
	}
	for i := range codes {
		code := codes[i]
		select {
		case <-ctx.Done():
			return errors.New("手动停止")
		default:
		}

		err = this.Pool.Do(func(c *tdx.Client) error {

			defer func() {
				plan(i+1, total)
			}()

			resp, err := c.GetMinuteTradeAll(code)
			if err != nil {
				return err
			}
			_ = resp

			m := [16]int{}
			for _, v := range resp.List {
				if v.Time > "09:45" {
					break
				}
				xs := strings.Split(v.Time, ":")
				if len(xs) == 2 {
					x := conv.Int(xs[1]) - 30
					if x >= 0 && x <= 14 {
						m[x] += v.Volume
					}
				}
			}

			logs.Debug(code)
			ls := []any{code, lastDate.Format("2006/01/02")}
			for _, v := range m {
				ls = append(ls, v)
			}
			lss = append(lss, ls)

			return nil
		})

		dealErr(code, err)

	}

	buf, err := excel.ToCsv(lss)
	if err != nil {
		return err
	}

	return oss.New(filepath.Join(this.Dir, lastDate.Format("2006-01-02"), "成交量.csv"), buf)
}
