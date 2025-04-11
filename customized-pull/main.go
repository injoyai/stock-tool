package main

import (
	"context"
	"customized-pull/api"
	_ "embed"
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/conv/codec"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"strings"
	"time"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.SetShowColor(false)
}

//go:embed index.html
var index string

func main() {

	err := lorca.Run(&lorca.Config{
		Width:  800,
		Height: 1300,
		Index:  index,
	}, func(app lorca.APP) error {

		//if time.Now().After(time.Date(2025, 4, 15, 0, 0, 0, 0, time.Local)) {
		//	app.Eval(`log('试用过期')`)
		//	return nil
		//}

		configPath := "./config/config.json"
		codePath := "./股票列表.txt"
		oss.NewNotExist(configPath, g.Map{
			"clients":     1,
			"disks":       10,
			"dir":         "./data/",
			"codes":       []string{"sz000001"},
			"timeout":     2,
			"userText":    false,
			"avgDecimal":  "2",
			"avg2Scale":   "1",
			"avg2Decimal": "0",
			"minute1Day":  "1",
			"minute5Day":  "2",
			"minute15Day": "2",
			"minute30Day": "3",
			"hourDay":     "4",
			"dayDay":      "30",
			"Files":       "6000",
			//"interval": 100,
			//"start1":   "09:30",
			//"end1":     "11:30",
			//"start2":   "13:00",
			//"end2":     "15:00",
			//"auto":     false,
		})
		oss.NewNotExist(codePath, "")
		cfg.Init(cfg.WithFile(configPath, codec.Json))

		dealErr := func(err error) {
			if err != nil {
				logs.Err(err)
				app.Eval(fmt.Sprintf(`log('%s')`, err.Error()))
				return
			}
			app.Eval(`log('完成')`)
		}
		log := func(s string) { app.Eval(fmt.Sprintf(`log('%s')`, s)) }
		plan := func(current, total int) {
			app.Eval(fmt.Sprintf(`updateProgress(%d,%d)`, current, total))
		}
		ctx, cancel := context.WithCancel(context.Background())
		getCodes := func() ([]string, error) {
			if cfg.GetBool("useText") {
				str, err := oss.ReadString(codePath)
				if err != nil {
					return nil, err
				}
				return strings.Split(str, "\r\n"), nil
			}
			return cfg.GetStrings("codes"), nil
		}

		//连接服务器
		c := api.Dial(
			cfg.GetInt("clients", 1),
			cfg.GetInt("disks", 10),
			time.Duration(cfg.GetInt("timeout", 2))*time.Second,
			log,
		)
		log(fmt.Sprintf(`连接服务器成功...`))
		c.GetCodes = getCodes
		c.Dir = cfg.GetString("dir")

		app.Bind("_download_history", func(lastDateStr string) {
			lastDate, err := time.Parse("2006-01-02", lastDateStr)
			if err != nil {
				dealErr(err)
				return
			}
			failCodes := []string(nil)
			dealErr(c.Pull(ctx, lastDate, log, plan, func(code string, err error) {
				failCodes = append(failCodes, code)
			}, [6]int{
				cfg.GetInt("minute1Day", 1),
				cfg.GetInt("minute5Day", 2),
				cfg.GetInt("minute15Day", 2),
				cfg.GetInt("minute30Day", 3),
				cfg.GetInt("hourDay", 4),
				cfg.GetInt("dayDay", 30),
			},
				cfg.GetInt("avgDecimal", 2),
				cfg.GetInt("avg2Scale", 1),
				cfg.GetInt("avg2Decimal", 2),
				cfg.GetInt("Files", 6000),
			))
			if len(failCodes) > 0 {
				oss.New("./失败代码.txt", strings.Join(failCodes, "\r\n"))
			}
		})

		stop := func() {
			logs.Debug("停止下载...")
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			log("停止成功...")
		}

		app.Bind("_stop_download", stop)

		app.Bind("_get_config", func() string {
			logs.Debug(cfg.GetString(""))
			return cfg.GetString("")
		})

		app.Bind("_save_config", func(clients, disks, dir, timeout string, codes []string, useText bool, avgDecimal, avg2Scale, avg2Decimal, minute1Day, minute5Day, minute15Day, minute30Day, hourDay, dayDay string) {
			c.Dir = dir
			m := g.Map{
				"clients": clients,
				"disks":   disks,
				"dir":     dir,
				"timeout": timeout,
				"codes":   codes,
				"useText": useText,

				"avg2Scale":   avg2Scale,
				"avg2Decimal": avg2Decimal,
				"avgDecimal":  avgDecimal,
				"minute1Day":  minute1Day,
				"minute5Day":  minute5Day,
				"minute15Day": minute15Day,
				"minute30Day": minute30Day,
				"hourDay":     hourDay,
				"dayDay":      dayDay,

				//"interval": interval,
				//"start1":   startTime1,
				//"end1":     endTime1,
				//"start2":   startTime2,
				//"end2":     endTime2,
				//"auto":     auto,
			}
			dealErr(oss.New(configPath, m))
			cfg.Init(cfg.WithAny(m))
			c.GetCodes = getCodes
			//if auto {
			//	refresh(false)
			//}
		})

		return nil
	})

	logs.PrintErr(err)
}
