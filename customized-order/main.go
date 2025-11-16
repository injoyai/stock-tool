package main

import (
	_ "embed"
	"fmt"
	"github.com/injoyai/base/types"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"math"
	"sync"
	"time"
)

//go:embed index.html
var html string

var (
	Filename  = cfg.GetString("filename", "./data/20060102-150405.csv")
	Coroutine = cfg.GetInt("coroutine", 10)
	Specs     = cfg.GetStrings("spec")
)

func main() {

	g.RecoverPrint(true)

	err := lorca.Run(&lorca.Config{
		Width:  800,
		Height: 600,
		Index:  html,
	}, func(app lorca.APP) error {

		var m *tdx.Manage

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
		{"代码", "名称", "现价", "涨跌幅", "成交额", "成交量", "总成交笔数", "现量",
			"卖五", "卖四", "卖三", "卖二", "卖一", "收盘笔数",
			"卖五量", "卖四量", "卖三量", "卖二量", "卖一量",
			"买一量", "买二量", "买三量", "买四量", "买五量",
			"买一", "买二", "买三", "买四", "买五",
			"今开", "最高", "最低", "开盘量", "开盘笔数", "委买量", "委卖量", "委差", "委加",
		},
	}
	start := time.Date(0, 0, 0, 9, 31, 0, 0, time.Local)
	end := time.Date(0, 0, 0, 11, 30, 0, 1, time.Local)
	for i := start; i.Before(end); i = i.Add(time.Minute) {
		data[0] = append(data[0], i.Format("1504量"))
	}
	start = time.Date(0, 0, 0, 13, 1, 0, 0, time.Local)
	end = time.Date(0, 0, 0, 15, 0, 0, 1, time.Local)
	for i := start; i.Before(end); i = i.Add(time.Minute) {
		data[0] = append(data[0], i.Format("1504量"))
	}
	for _, v := range quotes {
		code := v.Exchange.String() + v.Code
		totalBuy := v.BuyLevel[0].Number + v.BuyLevel[1].Number + v.BuyLevel[2].Number + v.BuyLevel[3].Number + v.BuyLevel[4].Number
		totalSell := v.SellLevel[0].Number + v.SellLevel[1].Number + v.SellLevel[2].Number + v.SellLevel[3].Number + v.SellLevel[4].Number
		ls := []any{
			code, cs.GetName(code), v.K.Close.Float64(), (v.K.Close - v.K.Open).Float64(), v.Amount, v.TotalHand, mTradeNumber[code], v.Intuition,
			v.SellLevel[4].Price.Float64(), v.SellLevel[3].Price.Float64(), v.SellLevel[2].Price.Float64(), v.SellLevel[1].Price.Float64(), v.SellLevel[0].Price.Float64(), mTradeLast[code],
			v.SellLevel[4].Number, v.SellLevel[3].Number, v.SellLevel[2].Number, v.SellLevel[1].Number, v.SellLevel[0].Number,
			v.BuyLevel[0].Number, v.BuyLevel[1].Number, v.BuyLevel[2].Number, v.BuyLevel[3].Number, v.BuyLevel[4].Number,
			v.BuyLevel[0].Price.Float64(), v.BuyLevel[1].Price.Float64(), v.BuyLevel[2].Price.Float64(), v.BuyLevel[3].Price.Float64(), v.BuyLevel[4].Price.Float64(),
			v.K.Open.Float64(), v.K.High.Float64(), v.K.Low.Float64(), mTradeFirstVol[code], mTradeFirst[code], totalBuy, totalSell, int64(math.Abs(float64(totalBuy - totalSell))), totalBuy + totalSell,
		}

		ts := mTrade[code]
		for _, vv := range ts.Klines() {
			ls = append(ls, vv.Volume)
		}

		data = append(data, ls)

	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}

	err = oss.New(filename, buf)

	return err
}
