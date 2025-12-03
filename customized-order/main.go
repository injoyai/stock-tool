package main

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/injoyai/base/types"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
)

//go:embed index.html
var html string

var (
	Filename   = cfg.GetString("filename", "./data/20060102-150405.csv")
	Coroutine  = cfg.GetInt("coroutine", 10)
	Specs      = cfg.GetStrings("spec")
	OrderTime  = cfg.GetStrings("orderTime")
	OrderTime2 = cfg.GetStrings("orderTime2")
)

func main() {

	g.RecoverPrint(true)

	err := lorca.Run(&lorca.Config{
		Width:  800,
		Height: 600,
		Index:  html,
	}, func(app lorca.APP) error {

		var m *tdx.Manage
		app.Eval("setDownloading(true)")
		defer app.Eval("setDownloading(false)")

		err := app.Bind("download", func() {

			if m == nil {
				defer app.Eval("appendLog(`正在连接服务器,请稍后再试...`)")
			}

			app.Eval("setProgress(0)")
			defer app.Eval("setDownloading(false)")
			app.Eval("setDownloading(true)")

			now := time.Now()

			codes := types.List[string](m.Codes.GetStocks())

			codesList := codes.Split(80)

			quotes := []*protocol.Quote(nil)

			wg := sync.WaitGroup{}
			for i := range codesList {
				ls := codesList[i]
				wg.Add(1)
				m.Go(func(c *tdx.Client) {
					defer wg.Done()
					resp, err := c.GetQuote(ls...)
					if err != nil {
						app.Eval(fmt.Sprintf("appendLog(`[错误]: %s`)", err))
						return
					}
					quotes = append(quotes, resp...)
				})
			}

			app.Eval("appendLog(`[信息] 拉取五档报价成功...`)")

			mTradeNumber := map[string]int{}
			mTradeFirstVol := map[string]int{}
			mTradeFirst := map[string]int{}
			mTradeLast := map[string]int{}
			mTrade := map[string]protocol.Trades{}
			mu := sync.Mutex{}
			current := 0
			for i := range codes {
				code := codes[i]
				wg.Add(1)
				m.Go(func(c *tdx.Client) {
					defer wg.Done()
					defer func() {
						current++
						app.Eval(fmt.Sprintf("setProgress(%d)", int(float64(current)/float64(codes.Len())*100)))
					}()
					resp, err := c.GetTradeAll(code)
					if err != nil {
						app.Eval(fmt.Sprintf("appendLog(`[错误]: %s`)", err))
						return
					}
					n := 0
					first := 0
					firstVol := 0
					last := 0
					for ii, v := range resp.List {
						n += v.Number
						last = v.Number
						if ii == 0 {
							first = v.Number
							firstVol = v.Volume
						}
					}
					mu.Lock()
					mTradeNumber[code] = n
					mTradeLast[code] = last
					mTradeFirst[code] = first
					mTradeFirstVol[code] = firstVol
					mTrade[code] = resp.List
					mu.Unlock()
				})

			}

			app.Eval("appendLog(`[信息] 拉取成交笔数成功...`)")

			wg.Wait()

			filename := now.Format(Filename)
			err := toCsv(filename, quotes, m.Codes, mTradeNumber, mTradeLast, mTradeFirst, mTradeFirstVol, mTrade)
			if err != nil {
				app.Eval(fmt.Sprintf("appendLog(`[错误]: %s`)", err))
				return
			}

			app.Eval("appendLog(`[信息] 导出成功`)")
		})
		if err != nil {
			logs.Err(err)
			app.Eval(fmt.Sprintf("appendLog(`[错误] 绑定函数失败: %s`)", err))
			return err
		}

		m, err = tdx.NewManage(&tdx.ManageConfig{Number: Coroutine})
		if err != nil {
			logs.Err(err)
			app.Eval(fmt.Sprintf("appendLog(`[错误] 连接服务器失败: %s`)", err))
			return err
		}

		app.Eval("appendLog(`[信息] 连接服务器成功...`)")

		cr := cron.New(cron.WithSeconds())
		for _, spec := range Specs {
			_, err = cr.AddFunc(spec, func() { app.Eval("download()") })
			if err != nil {
				logs.Err(err)
				app.Eval(fmt.Sprintf("appendLog(`[错误] 添加定时任务失败: %s`)", err))
				continue
			}
		}

		cr.Start()

		return nil
	})
	logs.PrintErr(err)
}

func toCsv(filename string, quotes protocol.QuotesResp, cs *tdx.Codes, mTradeNumber, mTradeLast, mTradeFirst, mTradeFirstVol map[string]int, mTrade map[string]protocol.Trades) error {
	data := [][]any{
		{"代码", "名称", "现价", "涨跌幅", "成交额", "成交量", "总成交笔数", "现量", "收盘笔数",
			"卖五", "卖四", "卖三", "卖二", "卖一",
			"卖五量", "卖四量", "卖三量", "卖二量", "卖一量",
			"买一量", "买二量", "买三量", "买四量", "买五量",
			"买一", "买二", "买三", "买四", "买五",
			"今开", "最高", "最低", "开盘量", "开盘笔数", "委买量", "委卖量", "委差", "委加",
		},
	}

	//成交量
	start := time.Date(0, 0, 0, 9, 30, 0, 0, time.Local)
	end := time.Date(0, 0, 0, 11, 30, 0, 0, time.Local)
	for i := start; i.Before(end); i = i.Add(time.Minute) {
		data[0] = append(data[0], i.Format("1504量"))
	}
	start = time.Date(0, 0, 0, 13, 0, 0, 0, time.Local)
	end = time.Date(0, 0, 0, 15, 0, 0, 0, time.Local)
	for i := start; i.Before(end); i = i.Add(time.Minute) {
		data[0] = append(data[0], i.Format("1504量"))
	}

	data[0] = append(data[0], "备份1", "备份2", "备份3", "备份4", "备份5", "备份6", "0925分笔")
	//
	titles := []string(nil)
	for _, v := range OrderTime2 {
		titles = append(titles, v)
		for x := range 20 {
			data[0] = append(data[0], v+fmt.Sprintf("%02d分笔", x+1))
		}
	}

	data[0] = append(data[0], "1457分笔", "1500分笔")
	data[0] = append(data[0], "备份7", "备份8", "备份9", "备份10", "备份11", "备份12")

	orderMap := map[string]bool{"0925": true, "0930": true}
	data[0] = append(data[0], "930笔数")
	for _, v := range OrderTime {
		data[0] = append(data[0], v+"笔数")
		orderMap[v] = true
	}

	//data[0] = append(data[0], "930笔数", "931笔数", "932笔数", "933笔数", "934笔数", "1453笔数", "1454笔数", "1455笔数", "1456笔数", "1457笔数", "1458笔数", "1459笔数", "1300笔数")

	for _, v := range quotes {
		code := v.Exchange.String() + v.Code
		totalBuy := v.BuyLevel[0].Number + v.BuyLevel[1].Number + v.BuyLevel[2].Number + v.BuyLevel[3].Number + v.BuyLevel[4].Number
		totalSell := v.SellLevel[0].Number + v.SellLevel[1].Number + v.SellLevel[2].Number + v.SellLevel[3].Number + v.SellLevel[4].Number
		ls := []any{
			_code(code), cs.GetName(code), int64(v.K.Close / 10), (v.K.Close - v.K.Open).Float64(), v.Amount, v.TotalHand, mTradeNumber[code], v.Intuition, mTradeLast[code],
			v.SellLevel[4].Price.Float64(), v.SellLevel[3].Price.Float64(), v.SellLevel[2].Price.Float64(), v.SellLevel[1].Price.Float64(), v.SellLevel[0].Price.Float64(),
			v.SellLevel[4].Number, v.SellLevel[3].Number, v.SellLevel[2].Number, v.SellLevel[1].Number, v.SellLevel[0].Number,
			v.BuyLevel[0].Number, v.BuyLevel[1].Number, v.BuyLevel[2].Number, v.BuyLevel[3].Number, v.BuyLevel[4].Number,
			v.BuyLevel[0].Price.Float64(), v.BuyLevel[1].Price.Float64(), v.BuyLevel[2].Price.Float64(), v.BuyLevel[3].Price.Float64(), v.BuyLevel[4].Price.Float64(),
			int64(v.K.Open / 10), int64(v.K.High / 10), int64(v.K.Low / 10), mTradeFirstVol[code], mTradeFirst[code], totalBuy, totalSell, int64(math.Abs(float64(totalBuy - totalSell))), totalBuy + totalSell,
		}

		ts := mTrade[code]
		for _, vv := range ts.Klines() {
			if vv.Volume == 0 {
				ls = append(ls, "")
				continue
			}
			ls = append(ls, vv.Volume)
		}

		ls = append(ls, "", "", "", "", "", "") //备份1-6
		{
			m := map[string]protocol.Trades{}
			for _, vv := range ts {
				s := vv.Time.Format("1504")
				m[s] = append(m[s], vv)
			}
			if val, ok := m["0925"]; ok && len(val) == 1 {
				ls = append(ls, toString(val[0]))
			} else {
				ls = append(ls, toString(nil))
			}
			for _, title := range titles {
				ss := m[title]
				for len(ss) < 20 {
					ss = append(ss, nil)
				}
				for _, vv := range ss {
					ls = append(ls, toString(vv))
				}
			}
			if val, ok := m["1457"]; ok && len(val) == 1 {
				ls = append(ls, toString(val[0]))
			} else {
				ls = append(ls, toString(nil))
			}
			if val, ok := m["1500"]; ok && len(val) == 1 {
				ls = append(ls, toString(val[0]))
			} else {
				ls = append(ls, toString(nil))
			}
		}
		ls = append(ls, "", "", "", "", "", "") //备份7-12

		{
			m := map[string]int{}
			for _, vv := range ts {
				s := vv.Time.Format("1504")
				if orderMap[s] {
					m[s] = m[s] + vv.Number
				}
			}
			ls = append(ls, m["0925"]+m["0930"])
			for _, s := range OrderTime {
				ls = append(ls, m[s])
			}
		}

		data = append(data, ls)

	}

	//types.List[[]any](data).Sort(func(a, b []any) bool {
	//	if len(a) == 0 || len(b) == 0 {
	//		return false
	//	}
	//	if conv.String(a[0]) == "代码" {
	//		return true
	//	}
	//	return conv.String(a[0]) < conv.String(b[0])
	//})

	sort.Slice(data[1:], func(i, j int) bool {
		if len(data[i+1]) == 0 || len(data[j+1]) == 0 {
			return false
		}
		return conv.String(data[i+1][0]) < conv.String(data[j+1][0])
	})

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}

	err = oss.New(filename, buf)

	return err
}

func _code(code string) string {
	if len(code) == 8 {
		return code[2:] + "." + code[:2]
	}
	return code
}

func toString(vv *protocol.Trade) any {
	if vv == nil {
		return ""
	}
	return fmt.Sprintf("%d_%d_%d_%d",
		int(vv.Price/10),
		int((vv.Price.Float64()*float64(vv.Volume)+5)/10),
		vv.Volume,
		vv.Number,
	)
}
