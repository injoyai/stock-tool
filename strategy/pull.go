package strategy

import (
	"context"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"path/filepath"
	"time"
)

func Pull() error {
	pull := extend.NewPullKline(extend.PullKlineConfig{
		Codes:   nil,
		Tables:  []string{extend.Day, extend.Month},
		Dir:     filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
		Limit:   1,
		StartAt: time.Time{},
	})

	m, err := tdx.NewManage(&tdx.ManageConfig{})
	if err != nil {
		return err
	}

	err = pull.Run(context.Background(), m)
	return err
}
