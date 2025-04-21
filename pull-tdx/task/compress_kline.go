package task

import (
	"context"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/tdx"
	"os"
	"path/filepath"
)

func NewCompressKline(csvDir, uploadDir string, tables map[string]string) *CompressKline {
	return &CompressKline{
		CsvDir:    csvDir,
		UploadDir: uploadDir,
		Tables:    tables,
	}
}

type CompressKline struct {
	CsvDir    string
	UploadDir string
	Tables    map[string]string //需要导出的表
}

func (this *CompressKline) Name() string {
	return "压缩k线"
}

func (this *CompressKline) Run(ctx context.Context, m *tdx.Manage) error {
	//生成压缩文件
	os.MkdirAll(this.UploadDir, 0777)
	r := &Range{
		Codes: func() []string {
			ls := []string(nil)
			for _, tableName := range this.Tables {
				ls = append(ls, tableName)
			}
			return ls
		}(),
		Limit:   len(this.Tables),
		Retry:   3,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *CompressKline) Handler(ctx context.Context, m *tdx.Manage, tableName string) error {
	from := filepath.Join(this.CsvDir, tableName)
	to := filepath.Join(this.UploadDir, tableName+".zip")
	return zip.Encode(from, to)
}
