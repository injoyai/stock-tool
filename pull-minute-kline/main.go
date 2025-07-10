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
	Spec        = cfg.GetString("spec", "0 10 15 * * *")
	Codes       = cfg.GetStrings("codes")
	Startup     = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.10")
	logs.Info("说明:", "修复并发没有限制的bug")
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
		).Run(context.Background(), m)
		logs.PrintErr(err)

		logs.Info("导出增量...")
		err = NewPullByDay(
			Codes,
			Coroutines,
			filepath.Join(ExportDir, "day"),
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
