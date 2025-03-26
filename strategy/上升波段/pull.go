package main

import (
	"context"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"path/filepath"
)

func main1() {
	pull := extend.NewPullKline(
		nil,
		[]string{extend.Day},
		filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
		1,
	)

	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)

	err = pull.Run(context.Background(), m)
	logs.Err(err)
}
