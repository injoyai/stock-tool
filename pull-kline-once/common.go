package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/notice"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

type Handler func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error)

func pull(codes []string, start, end time.Time, f func(c *tdx.Client) Handler) (map[string][]*protocol.Kline, error) {

	c, err := tdx.DialDefault()
	if err != nil {
		return nil, err
	}

	if len(codes) == 0 {
		codes = tdx.DefaultCodes.GetStocks()
	}

	result := make(map[string][]*protocol.Kline)

	handler := f(c)

	for i, code := range codes {
		startTime := time.Now()
		resp, err := handler(code, func(k *protocol.Kline) bool {
			return k.Time.Before(start)
		})
		logs.Debugf("序号: %d, 代码: %s. 耗时: %s, 结果: %v\n", i+1, code, time.Since(startTime), conv.New(err).String("成功"))
		if err != nil {
			logs.Err(err)
			continue
		}

		if len(resp.List) > 0 {
			resp.List = resp.List[1:]
		}

		ls := []*protocol.Kline(nil)
		for _, v := range resp.List {
			if v.Time.After(end) {
				break
			}
			ls = append(ls, v)
		}
		result[code] = ls

	}

	return result, nil
}

func done() func() {
	start := time.Now()
	return func() {
		notice.DefaultWindows.Publish(&notice.Message{Content: "执行结束,耗时: " + time.Since(start).String()})
	}
}

func body(code string, v *protocol.Kline) []any {
	return []any{
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
	}
}

var title = []any{"时间", "代码", "名称", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "涨幅", "涨幅比"}
