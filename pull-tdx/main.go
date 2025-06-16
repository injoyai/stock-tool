package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv/cfg"
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
	Version = "v0.5"
	Details = "增加复权数据"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	dirExport   = filepath.Join(dirBase, cfg.GetString("dir.export", "export"))
	dirUpload   = filepath.Join(dirBase, cfg.GetString("dir.upload", "upload"))
	clients     = cfg.GetInt("clients", 10)
	sendKey     = cfg.GetString("notice.serverChan.sendKey")
	config      = &tdx.ManageConfig{Number: clients}
	disks       = cfg.GetInt("disks", 150)
	spec        = cfg.GetString("spec", "0 1 15 * * *")
	specFQ      = "0 0 6 * * *"
	codes       = cfg.GetStrings("codes")
	startup     = cfg.GetBool("startup")
)

var (
	dirDatabaseKline       = filepath.Join(dirDatabase, "kline")
	dirDatabaseTrade       = filepath.Join(dirDatabase, "trade")
	dirExportKline         = filepath.Join(dirExport, "k线")
	dirExportCompressKline = filepath.Join(dirExport, "压缩/k线")
	dirUploadKline         = filepath.Join(dirUpload, "k线")
	dirUploadIndex         = filepath.Join(dirUpload, "指数")
	dirIncrementKline      = filepath.Join(dirUpload, "增量")
	dirUploadTrade         = filepath.Join(dirUpload, "分时成交")
	dirExportTrade         = filepath.Join(dirExport, "分时成交")
)

var (
	tasks = []task.Tasker{

		////指数
		//task.NewPullIndex(dirUploadIndex, nil),
		//
		////增量
		//task.NewPullKlineDay(codes, dirIncrementKline),

		//k线
		task.Group("k线",
			task.NewPullKline(codes, dirDatabaseKline, disks),                                   //拉取
			task.NewExportKline(codes, dirDatabaseKline, dirExportKline, disks, task.AllTables), //导出
			task.NewCompressKline(dirExportKline, dirExportCompressKline, task.AllTables),       //压缩
			task.NewRename(dirExportCompressKline, dirUploadKline),                              //移动
			task.NewNoticeServerChan(sendKey, "k线同步完成"),
		),

		task.Group("分时成交",
			//task.NewPullTradeHistory(codes, dirExportTrade, disks), //拉取
			task.NewPullTrade(codes, dirDatabaseTrade, disks),                   //拉取
			task.NewExportTrade(codes, dirDatabaseTrade, dirUploadTrade, disks), //导出
			task.NewNoticeServerChan(sendKey, "分时成交同步完成"),
		),
	}

	tasksFQ = []task.Tasker{
		task.NewPullKlineFQ(codes, dirExportKline),                                    //拉取复权数据
		task.NewExportKlineFQ(dirExportKline, dirExportCompressKline, dirUploadKline), //压缩移动
		task.NewNoticeServerChan(sendKey, "复权数据同步完成"),
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

	//1. 连接服务器
	m, err := tdx.NewManage(config, tdx.WithRedial())
	logs.PanicErr(err)

	/*



	 */

	//2. 任务内容
	f := func(tasks ...[]task.Tasker) func() {
		return func() {
			if !m.Workday.TodayIs() && !startup {
				logs.Err("今天不是工作日")
				return
			}
			fmt.Println("================================================================")
			logs.Info("开始执行...")
			ctx := context.Background()
			for _, v := range tasks {
				err = task.Run(ctx, m, v...)
				logs.PrintErr(err)
			}
			logs.Info("执行完成")
		}
	}

	//3. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, f(tasks))
	cr.AddFunc(specFQ, f(tasksFQ))
	cr.Start()

	//4. 启动便执行
	if startup {
		f(tasks, tasksFQ)()
	}

	select {}
}
