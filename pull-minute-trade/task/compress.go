package task

import (
	"context"
	"github.com/injoyai/goutil/oss/compress/zip"
)

type Compress struct {
	From, To string
}

func (this *Compress) Name() string {
	return "压缩数据"
}

func (this *Compress) Run(ctx context.Context) error {
	return zip.Encode(this.From, this.To)
}
