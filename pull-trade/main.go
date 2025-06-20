package main

import (
	"context"
	"fmt"
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
	Codes      = cfg.GetStrings("codes")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.1")
	logs.Info("说明:", "可以自定义代码")
	fmt.Println("=====================================================")
}

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		Codes,
		filepath.Join(tdx.DefaultDatabaseDir, "trade"),
		Coroutines,
	)

	s.Run(context.Background(), m)
}
