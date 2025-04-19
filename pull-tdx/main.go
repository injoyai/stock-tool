package main

import (
	"context"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"log"
	"path/filepath"
	"pull-tdx/task"
)

const (
	Version = "v0.1"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	dirExport   = filepath.Join(dirBase, cfg.GetString("dir.export", "export"))
	dirUpload   = filepath.Join(dirBase, cfg.GetString("dir.upload", "upload"))
	clients     = cfg.GetInt("number", 10)
	config      = &tdx.ManageConfig{Number: clients}
	disks       = cfg.GetInt("disks", 100)
	spec        = cfg.GetString("spec", "0 1 15 * * *")
	codes       = cfg.GetStrings("codes")
	startup     = cfg.GetBool("startup")
)

var (
	tasks = []task.Tasker{
		task.NewPullKline(codes, filepath.Join(dirDatabase, "kline"), disks),                                                                                   //拉取数据
		task.NewExportKline(codes, filepath.Join(dirDatabase, "kline"), filepath.Join(dirExport, "k线"), filepath.Join(dirUpload, "k线"), disks, task.AllTables), //导出数据
		task.NewPullIndex(filepath.Join(dirUpload, "指数"), nil),
	}
)

func init() {
	logs.DefaultFormatter.SetFlag(log.Ltime | log.Lshortfile)
	//logs.SetFormatter(logs.TimeFormatter)

	logs.Info("版本:", Version)
	logs.Debug("连接客户端数量:", clients)
	logs.Debug("释放协程数量:", disks)
	logs.Debug("配置的股票代码:", codes)
	fmt.Println("================================================================")
}

func main() {
	run()
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
		fmt.Println("================================================================")
		logs.Debug("开始执行...")
		queue.Add()
		defer queue.Done()

		ctx := context.Background()

		err = task.Run(ctx, m, tasks...)
		logs.PrintErr(err)

	}

	//3. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, f)
	cr.Start()

	//4. 启动便执行
	if startup {
		go f()
	}

	select {}
}
