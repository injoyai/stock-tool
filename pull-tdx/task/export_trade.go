package task

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"
)

func NewExportTrade(codes []string, databaseDir, exportDir string, limit int) *ExportTrade {
	return &ExportTrade{
		Codes:       codes,
		DatabaseDir: tradeDir(databaseDir),
		ExportDir:   exportDir,
		Limit:       limit,
	}
}

type ExportTrade struct {
	Codes       []string
	DatabaseDir tradeDir
	ExportDir   string
	Limit       int
}

func (this *ExportTrade) Name() string {
	return "xx"
	//return "导出成交数据"
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
	//取最新的年份进行导出
	year, filename := this.DatabaseDir.lastYear(code)
	b, err := db.Open(filename)
	if err != nil {
		return err
	}
	defer b.Close()

	all := []*model.Trade(nil)
	err = b.Find(all)
	if err != nil {
		return err
	}

	data := [][]any(nil)
	for _, v := range all {
		data = append(data, []any{
			v.Date,
			v.Time,
			v.Price,
			v.Volume,
			v.Status,
		})
	}

	buf, err := csv.Export(data)
	if err != nil {
		logs.Err(err)
		return err
	}

	filename = filepath.Join(this.ExportDir, code+"-"+conv.String(year)+".csv")
	return oss.New(filename, buf)
}
