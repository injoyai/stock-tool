package main

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/conv/codec"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/stock-tool/trade/api"
	"time"
)

//go:embed index.html
var index string

func main() {

	lorca.Run(&lorca.Config{
		Width:  800,
		Height: 680,
		Index:  index,
	}, func(app lorca.APP) error {

		//小后门
		if time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local).Before(time.Now()) {
			app.Eval(`log('试用结束!!!')`)
			return nil
		}

		configPath := "./config/config.json"
		oss.NewNotExist(configPath, g.Map{
			"clients": 1,
			"disks":   10,
			"dir":     "./",
			"codes":   []string{"sz000001"},
		})
		cfg.Init(cfg.WithFile(configPath, codec.Json))

		dealErr := func(err error) {
			if err != nil {
				app.Eval(fmt.Sprintf(`log('%s')`, err.Error()))
				return
			}
			app.Eval(`log('成功')`)
		}
		log := func(s string) { app.Eval(fmt.Sprintf(`log('%s')`, s)) }

		//连接服务器
		c := api.Dial(log)
		log(fmt.Sprintf(`连接服务器[%s]成功...`, c.GetKey()))
		c.Codes = cfg.GetStrings("codes")
		c.Dir = cfg.GetString("dir")

		app.Bind("_download_today", func() {
			dealErr(c.DownloadTodayAll())
		})

		app.Bind("_refresh_real", func() {
			dealErr(errors.New("未实现"))
		})

		app.Bind("_download_history", func(startDate, endDate string) {
			logs.Debug(startDate, endDate)
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
			dealErr(c.DownloadHistoryAll(start, end, log))
		})

		app.Bind("_stop_download", func() {
			dealErr(errors.New("未实现"))
		})

		app.Bind("_get_config", func() string {
			return cfg.GetString("")
		})

		app.Bind("_save_config", func(clientConnections, diskOperations, savePath string, stockCodes []string) {
			dealErr(oss.New(configPath, g.Map{
				"clients": clientConnections,
				"disks":   diskOperations,
				"dir":     savePath,
				"codes":   stockCodes,
			}))
			cfg.Init(cfg.WithFile(configPath))
		})

		return nil
	})

}
