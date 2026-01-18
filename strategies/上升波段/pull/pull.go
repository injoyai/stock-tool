package main

import (
	"context"
	"path/filepath"

	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
)

func main() {
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
