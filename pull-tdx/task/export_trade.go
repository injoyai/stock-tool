package task

import (
	"context"
	"github.com/injoyai/tdx"
)

func NewExportTrade(codes []string, dir string, limit int) *ExportTrade {
	return &ExportTrade{
		Codes: codes,
		Dir:   dir,
		Limit: limit,
	}
}

type ExportTrade struct {
	Codes []string
	Dir   string
	Limit int
}

func (this *ExportTrade) Name() string {
	return "导出成交数据"
}

func (this *ExportTrade) Run(ctx context.Context, m *tdx.Manage) error {
	r := &Range[string]{
		Codes:   GetCodes(m, this.Codes),
		Append:  nil,
		Limit:   this.Limit,
		Retry:   DefaultRetry,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *ExportTrade) Handler(ctx context.Context, m *tdx.Manage, code string) error {
	return nil
}
