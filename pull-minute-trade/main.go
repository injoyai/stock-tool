package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"log"
	"path/filepath"
	"pull-minute-trade/task"
)

const (
	Version = "v0.1"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	//minute1KlineDir   = cfg.GetString("dir", "./data/csv/1分钟K线/")
	//minute5KlineDir   = cfg.GetString("dir", "./data/csv/5分钟K线/")
	//dayKlineDir       = cfg.GetString("dir", "./data/csv/日K线/")
	//dayKlineByDateDir = cfg.GetString("dir", "./data/csv/日K线(按日期)/")
	config = &tdx.ManageConfig{
		Hosts:      cfg.GetStrings("hosts"),
		Number:     cfg.GetInt("number", 2),
		CodesDir:   dirDatabase,
		WorkdayDir: dirDatabase,
	}
	disks   = cfg.GetInt("disks", 1)
	spec    = cfg.GetString("spec", "0 1 15 * * *")
	codes   = cfg.GetStrings("codes")
	startup = cfg.GetBool("startup")
)

func init() {
	logs.DefaultFormatter.SetFlag(log.Ltime | log.Lshortfile)
	//logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", Version)
	logs.Debug("连接客户端数量:", cfg.GetInt("number", 2))
	logs.Debug("释放协程数量:", disks)
	logs.Debug("配置的股票代码:", codes)
}

func main() {
	_init()
	http(20001)
}

func _init() {

	//1. 连接服务器
	m, err := tdx.NewManage(config, tdx.WithRedial())
	logs.PanicErr(err)

	/*



	 */

	//2. 初始化
	queue := chans.NewLimit(1)
	f := func() {
		queue.Add()
		defer queue.Done()

		ctx := context.Background()

		tasks := []task.Tasker{
			task.NewPullTrade(m, codes, dirDatabase, disks),
			//plugins.NewExportMinuteKline(
			//	m,
			//	codes,
			//	filepath.Join(dir, "trade"),
			//	minute1KlineDir,
			//	minute5KlineDir,
			//	uint(disks),
			//),
		}

		err = task.Run(ctx, tasks...)
		logs.PrintErr(err)

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

func http(port int) error {
	s := mux.New()
	s.Group("/api", func(g *mux.Grouper) {
		g.POST("/task", func(r *mux.Request) {

		})
		g.POST("/execute", func(r *mux.Request) {

		})
	})
	s.SetPort(port)
	return s.Run()
}
