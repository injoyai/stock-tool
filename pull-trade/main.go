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
	Clients = cfg.GetInt("clients", 4)
	Disks   = cfg.GetInt("disks", 10)
	DSN     = cfg.GetString("database")
)

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		[]string{
			"sz000001", "sh600000",
		},
		filepath.Join(tdx.DefaultDatabaseDir, "trade"),
		Disks,
	)

	s.Run(context.Background(), m)
}
