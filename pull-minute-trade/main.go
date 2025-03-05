package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/goutil/oss/tray"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"log"
	"path/filepath"
	"pull-minute-trade/plugins"
	"pull-minute-trade/task"
)

const (
	Version = "v0.1"
)

var (
	dir               = cfg.GetString("dir", "./data/database/")
	minute1KlineDir   = cfg.GetString("dir", "./data/csv/1分钟K线/")
	minute5KlineDir   = cfg.GetString("dir", "./data/csv/5分钟K线/")
	dayKlineDir       = cfg.GetString("dir", "./data/csv/日K线/")
	dayKlineByDateDir = cfg.GetString("dir", "./data/csv/日K线(按日期)/")
	config            = &tdx.ManageConfig{
		Hosts:      cfg.GetStrings("hosts"),
		Number:     cfg.GetInt("number", 2),
		CodesDir:   dir,
		WorkdayDir: dir,
	}
	disks   = cfg.GetInt("disks", 1)
	spec    = cfg.GetString("spec", "0 1 15 * * *")
	codes   = cfg.GetStrings("codes")
	startup = cfg.GetBool("startup")
)

func init() {
	logs.DefaultFormatter.SetFlag(log.Ltime | log.Lshortfile)
	//logs.SetFormatter(logs.TimeFormatter)
}

func main() {
	gui(_init, http)
}

func gui(op ...tray.Option) {
	tray.Run(
		tray.WithLabel("版本: "+Version),
		tray.WithIco(IcoStock),
		tray.WithHint("数据拉取工具"),

		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),

		func(s *tray.Tray) {
			for _, v := range op {
				v(s)
			}
		},
	)
}

func _init(s *tray.Tray) {

	logs.Debug("配置的股票代码:", codes)

	//1. 连接服务器
	m, err := tdx.NewManage(config)
	logs.PanicErr(err)

	/*



	 */

	//2. 初始化
	queue := chans.NewLimit(1)
	f := func() {
		queue.Add()
		defer queue.Done()

		ctx := context.Background()

		//task.Run(ctx, plugins.NewPullTrade(m, codes, dir, disks))

		task.Run(ctx, plugins.NewExportMinuteKline(
			m,
			codes,
			filepath.Join(dir, "trade"),
			minute1KlineDir,
			minute5KlineDir,
			uint(disks),
		))

	}

	//3. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, f)
	cr.Start()

	//4. 启动便执行
	if startup {
		logs.Info("马上执行...")
		go f()
	}

}

func http(_ *tray.Tray) {
	s := mux.New()
	s.Group("/api", func(g *mux.Grouper) {
		g.POST("/task", func(r *mux.Request) {

		})
		g.POST("/execute", func(r *mux.Request) {

		})
	})
	go s.SetPort(20001).Run()
}
