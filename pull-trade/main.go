package main

import (
	"context"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
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
	DSN        = cfg.GetString("database")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v1.0")
	logs.Info("说明:", "第一版")
}

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		[]string{},
		filepath.Join(tdx.DefaultDatabaseDir, "trade"),
		Coroutines,
	)

	s.Run(context.Background(), m)
}
