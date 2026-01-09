package task

import (
	"context"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"

	"github.com/injoyai/tdx"
)

func NewExportKline(codes []string, databaseDir, csvDir string, disks int, tables map[string]string) *ExportKline {
	return &ExportKline{
		Codes:       codes,
		DatabaseDir: databaseDir,
		CsvDir:      csvDir,
		Limit:       disks,
		Tables:      tables,
	}
}

type ExportKline struct {
	Codes       []string          //自定义导出的代码
	DatabaseDir string            //数据来源
	CsvDir      string            //保存位置
	Limit       int               //协程数量
	Tables      map[string]string //需要导出的表
}

func (this *ExportKline) Name() string {
	return "导出k线"
}

func (this *ExportKline) Run(ctx context.Context, m *tdx.Manage) error {
	r := &Range[string]{
		Codes:   GetCodes(m, this.Codes),
		Limit:   this.Limit,
		Retry:   tdx.DefaultRetry,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *ExportKline) Handler(ctx context.Context, m *tdx.Manage, code string) error {
	filename := filepath.Join(this.DatabaseDir, code+".db")
	return db.WithOpen(filename, func(db *db.Sqlite) error {
		for table, tableName := range this.Tables {
			//获取数据
			all := []*model.Kline(nil)
			err := db.Table(table).Asc("Date").Find(&all)
			if err != nil {
				return err
			}

			switch table {
			case "DayKline":

			default:
			}

			//生成csv文件
			if err := klineToCsv2(code, all, filepath.Join(this.CsvDir, tableName, code+".csv"), m.Codes.GetName); err != nil {
				return err
			}
		}
		return nil
	})
}
