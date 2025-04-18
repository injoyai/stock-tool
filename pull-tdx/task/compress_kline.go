package task

import (
	"context"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/tdx"
)

type CompressKline struct {
	From, To string
}

func (this *CompressKline) Name() string {
	return "压缩k线数据"
}

func (this *CompressKline) Run(ctx context.Context, m *tdx.Manage) error {

	return zip.Encode(this.From, this.To)
}
