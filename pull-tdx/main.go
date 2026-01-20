package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"pull-tdx/task"
	"runtime"
	"time"

	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/robfig/cron/v3"
)

const (
	Version = "v1.3"
	Details = "改版数据结构,增加ETF日线,修复bug"
)

var (
	dirBase     = cfg.GetString("dir.base", "./data/")
	dirDatabase = filepath.Join(dirBase, cfg.GetString("dir.database", "database"))
	dirExport   = filepath.Join(dirBase, cfg.GetString("dir.export", "export"))
	dirUpload   = filepath.Join(dirBase, cfg.GetString("dir.upload", "upload"))
	clients     = cfg.GetInt("clients", 5)
	sendKey     = cfg.GetString("notice.serverChan.sendKey")
	goroutines  = cfg.GetInt("goroutines", 50)
	spec        = cfg.GetString("spec", "0 1 15 * * *")
	specFQ      = cfg.GetString("specFQ", "0 0 6 * * *")
	codes       = cfg.GetStrings("codes")
	startup     = cfg.GetBool("startup")
	address     = cfg.GetString("address", "http://192.168.1.103:20000")
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
			task.NewPullKline(codes, dirDatabaseKline, goroutines),                                   //拉取
			task.NewExportKline(codes, dirDatabaseKline, dirExportKline, goroutines, task.AllTables), //导出
			task.NewCompressKline(dirExportKline, dirExportCompressKline, task.AllTables),            //压缩
			task.NewRename(dirExportCompressKline, dirUploadKline),                                   //移动
			task.NewNoticeServerChan(sendKey, "k线同步完成"),
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
	logs.Debug("释放协程数量:", goroutines)
	logs.Debug("配置的股票代码:", codes)
	fmt.Println("================================================================")
}

func main() {

	//1. 连接服务器
	m, err := tdx.NewManage(
		tdx.WithClients(clients),
		tdx.WithCodes(nil),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) {
			return extend.DialCodesHTTP(address)
		}),
		tdx.WithGbbq(nil),
		tdx.WithDialGbbq(func(c *tdx.Client) (tdx.IGbbq, error) {
			return extend.DialGbbqHTTP(address)
		}),
	)
	logs.PanicErr(err)

	/*



	 */

	//2. 任务内容
	f := func(tasks ...[]task.Tasker) {
		fmt.Println("================================================================")
		logs.Info("开始执行...")
		ctx := context.Background()
		for _, v := range tasks {
			err = task.Run(ctx, m, v...)
			logs.PrintErr(err)
		}
		logs.Info("执行完成")
	}

	//3. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, func() {
		if !m.Workday.TodayIs() {
			logs.Err("今天不是工作日")
			return
		}
		f(tasks)
	})
	cr.AddFunc(specFQ, func() {
		if !m.Workday.Is(time.Now().AddDate(0, 0, -1)) {
			logs.Err("昨天不是工作日")
			return
		}
		f(tasksFQ)
	})

	//4. 启动便执行
	if startup {
		f(tasks, tasksFQ)
	}

	cr.Run()
}
