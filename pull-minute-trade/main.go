package main

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"pull-minute-trade/db"
)

var (
	dir    = cfg.GetString("dir", "./data")
	config = &tdx.ManageConfig{
		Hosts:  cfg.GetStrings("hosts"),
		Number: cfg.GetInt("number", 1),
		Dir:    dir,
	}
	disks = cfg.GetInt("disks", 1)
	spec  = cfg.GetString("spec", "0 1 15 * * *")
)

func main() {

	m, err := tdx.NewManage(config)
	logs.PanicErr(err)

	limitGet := chans.NewLimit(disks)
	limitSave := chans.NewLimit(disks)
	chanGet := make(chan *db.Message, 100)
	chanSave := make(chan *db.Message, 100)

	//1. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, func() {

		//获取所有股票代码
		codes := m.Codes.GetStocks()

		//2. 从服务器拉取数据
		go func() {
			for {
				select {
				case data := <-chanGet:

					if data.Updated() {
						continue
					}

					m.Go(func(c *tdx.Client) {

						data.RangeDate(func(date string) {
							c.GetHistoryMinuteTradeAll(date, data.Code)

						})

					})
				}
			}

		}()

		//3. 更新到数据库
		go func() {
			for {
				select {
				case data := <-chanSave:
					limitSave.Add()
					go func() {
						defer limitSave.Done()
						_ = data
					}()
				}
			}
		}()

		//1. 获取每只股票的最后数据,加入缓存
		for i := range codes {
			limitGet.Add()
			go func(code string) {
				defer limitGet.Done()
				last, err := db.Open(filepath.Join(dir, "database", code+".db")).GetLast()
				if err != nil {
					logs.Err(err)
					return
				}
				chanGet <- &db.Message{
					Code:  code,
					Model: last,
				}
			}(codes[i])

		}

	})

}
