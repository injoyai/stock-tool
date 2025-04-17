package main

import (
	"context"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"path/filepath"
	"time"
)

func Pull(tables []string) (*tdx.Manage, error) {
	pull := extend.NewPullKline(extend.PullKlineConfig{
		Codes:   nil,
		Tables:  tables,
		Dir:     filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
		Limit:   1,
		StartAt: time.Time{},
	})

	m, err := tdx.NewManage(&tdx.ManageConfig{})
	if err != nil {
		return nil, err
	}

	err = pull.Run(context.Background(), m)
	return m, err
}
