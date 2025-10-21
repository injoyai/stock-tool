package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"strings"
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
	pull()
}

func pull() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	for _, v := range m.Codes.GetStocks() {
		if strings.HasPrefix(v, "bj") {
			Codes = append(Codes, v)
		}
	}

	s := NewSqlite(
		Codes,
		filepath.Join(DatabaseDir, "trade"),
		Coroutines,
		Tasks,
	)

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() { s.Run(context.Background(), m) })
	if Startup {
		s.Run(context.Background(), m)
	}
	t.Run()
}
