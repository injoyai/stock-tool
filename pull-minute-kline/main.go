package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"time"
)

var (
	Clients     = cfg.GetInt("clients", 4)
	Coroutines  = cfg.GetInt("coroutines", 10)
	DatabaseDir = cfg.GetString("database", "./data/database")
	ExportDir   = cfg.GetString("export", "./data/export")
	UploadDir   = cfg.GetString("upload", "./data/upload")
	Spec        = cfg.GetString("spec", "0 10 15 * * *")
	Codes       = cfg.GetStrings("codes")
	Startup     = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.13")
	logs.Info("说明:", "升级tdx版本,可以获取京市数据")
	logs.Info("任务规则:", Spec)
	logs.Info("立马执行:", Startup)
	logs.Info("连接数量:", Clients)
	logs.Info("协程数量:", Coroutines)
	fmt.Println("=====================================================")

}

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	f := func() {

		if !m.Workday.TodayIs() {
			logs.Err("今天不是工作日,跳过任务...")
			return
		}

		logs.Info("更新数据...")
		err = NewUpdateKline(
			Codes,
			filepath.Join(DatabaseDir, "kline"),
			Coroutines,
		).Run(context.Background(), m)
		logs.PrintErr(err)

		logs.Info("导出数据...")
		err = NewExportKline(
			Codes,
			Coroutines,
			[]int{time.Now().Year()},
			filepath.Join(DatabaseDir, "kline"),
			filepath.Join(ExportDir, "year"),
			filepath.Join(UploadDir),
		).Run(context.Background(), m)
		logs.PrintErr(err)

		logs.Info("导出增量...")
		err = NewPullByDay(
			Codes,
			Coroutines,
			filepath.Join(ExportDir, "day"),
			filepath.Join(UploadDir, "每日数据"),
		).Run(m, time.Now())
		logs.PrintErr(err)

		logs.Info("任务完成...")
	}

	corn := cron.New(cron.WithSeconds())
	corn.AddFunc(Spec, f)

	if Startup {
		f()
	}

	corn.Run()
}
