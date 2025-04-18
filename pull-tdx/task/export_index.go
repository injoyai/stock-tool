package task

import (
	"context"
	"github.com/injoyai/tdx"
)

// ExportIndex 导出指数
type ExportIndex struct {
	*ExportKline
}

func (this *ExportIndex) Name() string {
	return "导出指数数据"
}

func (this *ExportIndex) Run(ctx context.Context, m *tdx.Manage) error {
	return this.ExportKline.Run(ctx, m)
}
