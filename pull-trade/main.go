package main

import (
	"context"
	"fmt"
	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"path/filepath"
)

const (
	DefaultRetry = 3
)

var (
	Clients     = cfg.GetInt("clients", 4)
	Coroutines  = cfg.GetInt("coroutines", 10)
	Tasks       = cfg.GetInt("tasks", 2)
	DatabaseDir = cfg.GetString("database", "./data/database")
	Spec        = cfg.GetString("spec", "0 10 15 * * *")
	Codes       = cfg.GetStrings("codes")
	Startup     = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.6")
	logs.Info("说明:", "增加更新进度条")
	logs.Info("任务规则:", Spec)
	logs.Info("立马执行:", Startup)
	logs.Info("连接数量:", Clients)
	logs.Info("协程数量:", Coroutines)
	fmt.Println("=====================================================")

}

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStocks()
	}

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() { run(m) })
	if Startup {
		run(m)
	}
	t.Run()
}

func run(m *tdx.Manage) {
	logs.PrintErr(pull(m))
	logs.PrintErr(exportThisYear(m))
}

func pull(m *tdx.Manage) error {
	s := NewSqlite(
		Codes,
		filepath.Join(DatabaseDir, "trade"),
		Coroutines,
		Tasks,
	)
	return s.Run(context.Background(), m)
}

func exportThisYear(m *tdx.Manage) error {

	b := bar.NewCoroutine(len(Codes), Coroutines)
	defer b.Close()

	for i := range Codes {
		code := Codes[i]
		b.Go(func() {
			_ = code
		})
	}

	return nil
}
