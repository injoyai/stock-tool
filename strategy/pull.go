package strategy

import (
	"context"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"path/filepath"
)

func Pull() error {
	pull := extend.NewPullKline(
		nil,
		[]string{extend.Day, extend.Month},
		filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
		1,
	)

	m, err := tdx.NewManage(&tdx.ManageConfig{})
	if err != nil {
		return err
	}

	err = pull.Run(context.Background(), m)
	return err
}
