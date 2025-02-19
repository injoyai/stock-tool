package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/conv/codec"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/stock-tool/trade/api"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var index string

func main() {

	lorca.Run(&lorca.Config{
		Width:  800,
		Height: 900,
		Index:  index,
	}, func(app lorca.APP) error {

		configPath := "./config/config.json"
		codePath := "./股票列表.txt"
		oss.NewNotExist(configPath, g.Map{
			"clients":  10,
			"disks":    100,
			"dir":      "./data/",
			"codes":    []string{"sz000001"},
			"userText": false,
			"interval": 100,
			"start1":   "09:30",
			"end1":     "11:30",
			"start2":   "13:00",
			"end2":     "15:00",
			"auto":     false,
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
			cfg.GetInt("clients", 10),
			cfg.GetInt("disks", 100),
			log,
		)
		log(fmt.Sprintf(`连接服务器成功...`))
		c.GetCodes = getCodes
		c.Dir = cfg.GetString("dir")

		app.Bind("_download_today", func() {
			dealErr(c.DownloadTodayAll2(ctx, log, plan))
		})

		var refreshLock sync.Mutex
		refresh := func() {
			if !refreshLock.TryLock() {
				log("正在实时刷新数据中...")
				return
			}
			cc := ctx
			defer func() {
				refreshLock.Unlock()
				log("结束实时刷新数据...")
			}()
			for {
				select {
				case <-cc.Done():
					return
				default:

					now := time.Now()
					date := now.Format("2006-01-02 ")

					start1, err := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("start1", "09:30"), time.Local)
					if err != nil {
						start1 = now
					}

					end1, err := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("end1", "11:30"), time.Local)
					if err != nil {
						end1 = now
					}

					start2, err := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("start2", "13:00"), time.Local)
					if err != nil {
						start2 = now
					}

					end2, err := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("end2", "15:00"), time.Local)
					if err != nil {
						end2 = now
					}

					if now.Before(start1) || now.After(end2) || (now.After(end1) && now.Before(start2)) {

						min := time.Second
						if sub := start1.Sub(now); sub > min {
							min = sub
						}

						if sub := start2.Sub(now); sub < min {
							min = sub
						}

						if now.Sub(end2) > 0 {
							min = time.Hour * 2
						}

						min /= 2
						if min < time.Second {
							min = time.Second
						}

						//log(now.Format("15:04") + ": 未到设置的时间,等待" + min.String())

						<-time.After(min)
						continue
					}

					log("实时刷新数据...")
					dealErr(c.DownloadTodayAll2(cc, log, plan))
					<-time.After(time.Duration(cfg.GetInt("interval", 1000)) * time.Millisecond)
				}

			}

		}
		app.Bind("_refresh_real", refresh)
		if cfg.GetBool("auto", false) {
			log("开启自动刷新...")
			go refresh()
		}

		app.Bind("_download_history", func(startDate, endDate string) {
			start, err := time.Parse("2006-01-02", startDate)
			if err != nil {
				dealErr(err)
				return
			}
			end, err := time.Parse("2006-01-02", endDate)
			if err != nil {
				dealErr(err)
				return
			}
			dealErr(c.DownloadHistoryAll(ctx, start, end, log, plan))
		})

		app.Bind("_stop_download", func() {
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			log("停止成功...")
		})

		app.Bind("_get_config", func() string {
			return cfg.GetString("")
		})

		app.Bind("_save_config", func(clients, disks, dir string, codes []string, useText, auto bool, interval, startTime1, endTime1, startTime2, endTime2 string) {
			c.Dir = dir
			m := g.Map{
				"clients":  clients,
				"disks":    disks,
				"dir":      dir,
				"codes":    codes,
				"useText":  useText,
				"interval": interval,
				"start1":   startTime1,
				"end1":     endTime1,
				"start2":   startTime2,
				"end2":     endTime2,
				"auto":     auto,
			}
			dealErr(oss.New(configPath, m))
			cfg.Init(cfg.WithAny(m))
			c.GetCodes = getCodes
		})

		return nil
	})

}
