package task

import (
	"context"
	"os"
	"path/filepath"

	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/tdx"
)

func NewExportKlineFQ(exportDir, compressDir, uploadDir string) *ExportKlineFQ {
	return &ExportKlineFQ{
		ExportDir:   exportDir,
		CompressDir: compressDir,
		UploadDir:   uploadDir,
	}
}

type ExportKlineFQ struct {
	ExportDir   string
	CompressDir string
	UploadDir   string
}

func (this *ExportKlineFQ) Name() string {
	return "压缩复权日线"
}

func (this *ExportKlineFQ) Run(ctx context.Context, m *tdx.Manage) error {
	r := &Range[string]{
		Codes:   []string{"日线_前复权", "日线_后复权"},
		Limit:   2,
		Retry:   tdx.DefaultRetry,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *ExportKlineFQ) Handler(ctx context.Context, m *tdx.Manage, name string) error {
	exportFilename := filepath.Join(this.ExportDir, name)
	compressFilename := filepath.Join(this.CompressDir, name+".zip")
	uploadFilename := filepath.Join(this.UploadDir, name+".zip")
	if err := zip.Encode(exportFilename, compressFilename); err != nil {
		return err
	}
	return os.Rename(compressFilename, uploadFilename)
}
