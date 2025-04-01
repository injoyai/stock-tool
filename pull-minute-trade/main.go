package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"log"
	"path/filepath"
	"pull-minute-trade/task"
	"time"
)

const (
	Version = "v0.1"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	dirExport   = filepath.Join(dirBase, cfg.GetString("dir.export", "export"))
	config      = &tdx.ManageConfig{
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

var (
	tasks = []task.Tasker{
		//task.NewPullTrade(codes, filepath.Join(dirDatabase, "trade"), disks),
		//task.NewPullKline(codes, filepath.Join(dirDatabase, "kline"), disks),
		//task.NewExportMinuteKline(
		//	codes,
		//	filepath.Join(dirDatabase, "trade"),
		//	filepath.Join(dirExport, "csv", "1分钟"),
		//	filepath.Join(dirExport, "csv", "5分钟"),
		//	disks,
		//	),
		&task.ExportMinuteKlineAll{
			Codes:       codes,
			Start:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
			End:         time.Date(2004, 12, 31, 0, 0, 0, 0, time.Local),
			DatabaseDir: filepath.Join(dirDatabase, "trade"),
			OutputDir:   filepath.Join(dirExport, "csv", "1分钟"),
			Limit:       disks,
		},
	}
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
	run()
	http(20001)
}

func run() {

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

		err = task.Run(ctx, m, tasks...)
		logs.PrintErr(err)
	}

	//3. 设置定时
	//cr := cron.New(cron.WithSeconds())
	//cr.AddFunc(spec, f)
	//cr.Start()

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
