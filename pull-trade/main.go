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
	StartDate    = time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local)
	DefaultRetry = 3
)

var (
	Clients    = cfg.GetInt("clients", 4)
	Coroutines = cfg.GetInt("coroutines", 10)
	Tasks      = cfg.GetInt("tasks", 2)
	DSN        = cfg.GetString("database")
	Spec       = cfg.GetString("spec", "0 1 19 * * *")
	Codes      = cfg.GetStrings("codes")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.2")
	logs.Info("说明:", "增加定时任务")
	fmt.Println("=====================================================")
}

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		Codes,
		filepath.Join(tdx.DefaultDatabaseDir, "trade"),
		Coroutines,
		Tasks,
	)

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() { s.Run(context.Background(), m) })
	t.Run()

}
