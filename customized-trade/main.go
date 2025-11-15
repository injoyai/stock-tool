package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/injoyai/conv/cfg"
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
		Height: 1000,
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
			time.Duration(cfg.GetInt("timeout", 2))*time.Second,
			log,
		)
		log(fmt.Sprintf(`连接服务器成功...`))
		c.GetCodes = getCodes
		c.Dir = cfg.GetString("dir")

		app.Bind("_download_today", func() {
			failCodes := []string(nil)
			dealErr(c.DownloadTodayAll2(ctx, log, plan, func(code string, err error) {
				failCodes = append(failCodes, code)
			}))
			if len(failCodes) > 0 {
				oss.New("./失败代码.txt", strings.Join(failCodes, "\r\n"))
			}
		})

		var refreshLock sync.Mutex
		refresh := func(hand bool) {
			if !refreshLock.TryLock() {
				log("正在实时刷新数据中...")
				return
			}
			failCodes := []string(nil)
			cc := ctx
			defer func() {
				refreshLock.Unlock()
				log("结束实时刷新数据...")
				if len(failCodes) > 0 {
					oss.New("./失败代码.txt", strings.Join(failCodes, "\r\n"))
				}
			}()

			f := func() {
				log("实时刷新数据...")
				dealErr(c.DownloadTodayAll2(cc, log, plan, func(code string, err error) {
					failCodes = append(failCodes, code)
				}))
				<-time.After(time.Duration(cfg.GetInt("interval", 1000)) * time.Millisecond)
			}

			for {
				select {
				case <-cc.Done():
					return

				default:

					if hand {
						f()
						continue
					}

					now := time.Now()
					date := now.Format("2006-01-02 ")

					start1, _ := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("start1"), time.Local)
					end1, _ := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("end1"), time.Local)
					if !start1.IsZero() && !end1.IsZero() {
						if now.After(start1) && now.Before(end1) {
							f()
							continue
						}
					}

					start2, _ := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("start2"), time.Local)
					end2, _ := time.ParseInLocation("2006-01-02 15:04", date+cfg.GetString("end2"), time.Local)
					if !start2.IsZero() && !end2.IsZero() {
						if now.After(start2) && now.Before(end2) {
							f()
							continue
						}
					}

					<-time.After(time.Second * 1)
				}

			}

		}
		stop := func() {
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			log("停止成功...")
		}

		if cfg.GetBool("auto", false) {
			log("开启自动刷新...")
			go refresh(false)
		}

		app.Bind("_refresh_real", func() {
			stop()
			for i := 0; i < 20; i++ {
				<-time.After(time.Millisecond * 500)
				if refreshLock.TryLock() {
					refreshLock.Unlock()
					break
				}
			}
			refresh(true)
		})

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

		app.Bind("_stop_download", stop)

		app.Bind("_get_config", func() string {
			return cfg.GetString("")
		})

		app.Bind("_save_config", func(clients, disks, dir, timeout string, codes []string, useText, auto bool, interval, startTime1, endTime1, startTime2, endTime2 string) {
			c.Dir = dir
			m := g.Map{
				"clients":  clients,
				"disks":    disks,
				"dir":      dir,
				"timeout":  timeout,
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
			if auto {
				refresh(false)
			}
		})

		return nil
	})

}
