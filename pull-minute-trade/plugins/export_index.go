package plugins

import "context"

// ExportIndex 导出指数
type ExportIndex struct {
	*ExportKline
}

func (this *ExportIndex) Name() string {
	return "导出指数数据"
}

func (this *ExportIndex) Run(ctx context.Context) error {
	return this.ExportKline.Run(ctx)
}
