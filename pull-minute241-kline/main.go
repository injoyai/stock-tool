package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
)

var (
	Title = []any{
		"代码",
		"日期",
		"开盘",
		"最高",
		"最低",
		"收盘",
		"成交量(股)",
		"成交额(元)",
		"涨跌(元)",
		"涨跌幅(%)",
		"换手率(%)",
		"流通股本(股)",
		"总股本(股)",
	}
	DefaultRetry = tdx.DefaultRetry
)

var (
	Clients     = cfg.GetInt("clients", 4)
	Coroutines  = cfg.GetInt("coroutines", 10)
	DatabaseDir = cfg.GetString("database", "./data/database/kline")
	ExportDir   = cfg.GetString("export", "./data/output/export")
	UploadDir   = cfg.GetString("upload", "./data/output/upload")
	Spec        = cfg.GetString("spec", "0 10 15 * * *")
	Codes       = cfg.GetStrings("codes")
	Startup     = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.3.0")
	logs.Info("说明:", "241条数据版本")
	logs.Info("任务规则:", Spec)
	logs.Info("立马执行:", Startup)
	logs.Info("连接数量:", Clients)
	logs.Info("协程数量:", Coroutines)
	fmt.Println("=====================================================")

}

func main() {
	m, err := tdx.NewManage(tdx.WithClients(Clients))
	logs.PanicErr(err)

	f := func() {

		year := conv.String(time.Now().Year())

		if !m.Workday.TodayIs() {
			logs.Err("今天不是工作日,跳过任务...")
			return
		}

		codes := Codes
		if len(codes) == 0 {
			codes = m.Codes.GetStockCodes()
		}

		logs.Info("导出每日数据...")
		err = PullByDay(
			m,
			time.Now(),
			codes,
			Coroutines,
			filepath.Join(ExportDir, "day"),
			filepath.Join(UploadDir, "每日数据"),
		)
		logs.PrintErr(err)

		logs.Info("更新数据...")
		err = Update(
			m,
			codes,
			year,
			Coroutines,
			DatabaseDir,
		)
		logs.PrintErr(err)

		logs.Info("导出数据...")
		err = Export(
			m.Gbbq,
			codes,
			Coroutines,
			year,
			DatabaseDir,
			filepath.Join(ExportDir, "year"),
			filepath.Join(UploadDir),
		)
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
