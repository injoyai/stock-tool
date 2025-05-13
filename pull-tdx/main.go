package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"log"
	"path/filepath"
	"pull-tdx/task"
	"runtime"
	"time"
)

const (
	Version = "v0.3"
	Details = "增加重命名任务,以适配自动同步的问题"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	dirExport   = filepath.Join(dirBase, cfg.GetString("dir.export", "export"))
	dirUpload   = filepath.Join(dirBase, cfg.GetString("dir.upload", "upload"))
	clients     = cfg.GetInt("clients", 10)
	config      = &tdx.ManageConfig{Number: clients}
	disks       = cfg.GetInt("disks", 150)
	spec        = cfg.GetString("spec", "0 1 15 * * *")
	codes       = cfg.GetStrings("codes")
	startup     = cfg.GetBool("startup")
)

var (
	dirDatabaseKline       = filepath.Join(dirDatabase, "kline")
	dirExportKline         = filepath.Join(dirExport, "k线")
	dirExportCompressKline = filepath.Join(dirExport, "压缩/k线")
	dirUploadKline         = filepath.Join(dirUpload, "k线")
	dirUploadIndex         = filepath.Join(dirUpload, "指数")
	dirIncrementKline      = filepath.Join(dirUpload, "增量")
)

var (
	tasks = []task.Tasker{

		//指数
		task.NewPullIndex(dirUploadIndex, nil),

		//增量
		task.NewPullKlineDay(codes, dirIncrementKline),

		//k线
		task.Group("k线",
			task.NewPullKline(codes, dirDatabaseKline, disks),                                   //拉取
			task.NewExportKline(codes, dirDatabaseKline, dirExportKline, disks, task.AllTables), //导出
			task.NewCompressKline(dirExportKline, dirExportCompressKline, task.AllTables),       //压缩
			task.NewRename(dirExportCompressKline, dirUploadKline),                              //移动
		),
	}
)

func init() {
	logs.DefaultFormatter.SetFlag(log.Ltime | log.Lshortfile)
	logs.SetFormatter(logs.TimeFormatter)
	logs.SetShowColor(runtime.GOOS != "windows")

	logs.Info("版本:", Version)
	logs.Info("日期:", time.Now().Format(time.DateOnly))
	logs.Info("说明:", Details)
	logs.Debug("启动立马执行:", startup)
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

	//2. 任务内容
	f := func() {
		if !m.Workday.TodayIs() && !startup {
			logs.Err("今天不是工作日")
			return
		}
		fmt.Println("================================================================")
		logs.Debug("开始执行...")
		ctx := context.Background()
		err = task.Run(ctx, m, tasks...)
		logs.PrintErr(err)
		logs.Debug("执行完成")
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
