package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg"
	inin "github.com/injoyai/goutil/frame/in/v3"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"pull-minute-trade/plugins"
)

var (
	dir      = cfg.GetString("dir", "./data")
	database = filepath.Join(dir, "database")
	config   = &tdx.ManageConfig{
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

	queue := chans.NewLimit(1)

	/*



	 */

	//1. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, func() {
		queue.Add()
		defer queue.Done()
		plugins.NewPullTrade(m, database, disks).Run(context.Background())
	})
	cr.Start()

	/*



	 */

	s := mux.New()
	s.Group("/api", func(g *mux.Grouper) {
		g.POST("/task", func(r *mux.Request) {

		})
		g.POST("/execute", func(r *mux.Request) {
			if !queue.Try() {
				inin.Fail("有任务正在执行")
			}
			defer queue.Done()

		})
	})
	s.SetPort(20001).Run()

}
