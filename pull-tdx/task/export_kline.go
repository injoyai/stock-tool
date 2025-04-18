package task

import (
	"context"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/tdx"
	"os"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"time"
)

func NewExportKline(codes []string, databaseDir, csvDir, uploadDir string, disks int, tables map[string]string) *ExportKline {
	return &ExportKline{
		Codes:       codes,
		DatabaseDir: databaseDir,
		CsvDir:      csvDir,
		UploadDir:   uploadDir,
		Limit:       disks,
		Tables:      tables,
	}
}

type ExportKline struct {
	Codes       []string          //自定义导出的代码
	DatabaseDir string            //数据来源
	CsvDir      string            //保存位置
	UploadDir   string            //
	Limit       int               //协程数量
	Tables      map[string]string //需要导出的表
}

func (this *ExportKline) Name() string {
	return "导出k线数据"
}

func (this *ExportKline) Run(ctx context.Context, m *tdx.Manage) error {
	return this.byCode(ctx, m)
}

func (this *ExportKline) byCode(ctx context.Context, m *tdx.Manage) error {
	r := &Range{
		Codes: this.Codes,
		Limit: this.Limit,
		Handler: func(code string) error {
			filename := filepath.Join(this.DatabaseDir, code+".db")
			return db.WithOpen(filename, func(db *db.Sqlite) error {
				for table, tableName := range this.Tables {
					//获取数据
					all := []*model.Kline(nil)
					err := db.Table(table).Asc("Date").Find(&all)
					if err != nil {
						return err
					}

					//生成数据
					data := [][]any{title}
					for _, v := range all {
						t := time.Unix(v.Date, 0)
						data = append(data, []any{
							t.Format("2006-01-02"), t.Format("15:04"), code, m.Codes.GetName(code),
							v.Open.Float64(), v.Close.Float64(), v.High.Float64(), v.Low.Float64(), v.Volume, v.Amount.Float64(), v.RisePrice().Float64(), v.RiseRate(),
						})
					}
					buf, err := excel.ToCsv(data)
					if err != nil {
						return err
					}
					//生成csv
					if err = oss.New(filepath.Join(this.CsvDir, tableName, code+".csv"), buf); err != nil {
						return err
					}
					//生成压缩
					os.MkdirAll(this.UploadDir, 0777)
					if err = zip.Encode(filepath.Join(this.CsvDir, tableName), filepath.Join(this.UploadDir, tableName+".zip")); err != nil {
						return err
					}
				}
				return nil
			})

		},
	}
	return r.Run(ctx, m)
}
