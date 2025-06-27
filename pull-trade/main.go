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
	Clients     int
	Coroutines  int
	Tasks       int
	DatabaseDir string
	ExportDir   string
	Spec        string
	Codes       []string
)

func init() {
	cfg.Init(cfg.WithFile("./config/convert.yaml"))
	Clients = cfg.GetInt("clients", 4)
	Coroutines = cfg.GetInt("coroutines", 10)
	Tasks = cfg.GetInt("tasks", 2)
	DatabaseDir = cfg.GetString("database", "./data/database")
	ExportDir = cfg.GetString("export", "./data/export")
	Spec = cfg.GetString("spec", "0 1 19 * * *")
	Codes = cfg.GetStrings("codes")

	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.2")
	logs.Info("说明:", "增加定时任务")
	fmt.Println("=====================================================")
}

func main() {
	logs.Debug(Codes)
	convert()
}

func convert() {
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)
	c := NewConvert(
		Codes,
		"",
		filepath.Join(DatabaseDir, "trade"),
		filepath.Join(DatabaseDir, "kline_append1"),
		filepath.Join(DatabaseDir, "kline_append2"),
		filepath.Join(DatabaseDir, "kline"),
		time.Date(2024, 7, 21, 0, 0, 0, 0, time.Local),
	)
	c.Run(context.Background(), m)
}

func export() {
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)
	e := NewExport(
		[]string{"sz000001"},
		filepath.Join(DatabaseDir, "trade"),
		ExportDir,
	)
	e.Run(context.Background(), m)
}

func pull() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		Codes,
		filepath.Join(DatabaseDir, "trade"),
		Coroutines,
		Tasks,
	)

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() { s.Run(context.Background(), m) })
	t.Run()
}
