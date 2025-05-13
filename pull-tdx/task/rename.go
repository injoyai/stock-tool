package task

import (
	"context"
	"github.com/injoyai/tdx"
	"os"
	"path/filepath"
)

func NewRename(source, target string) *Rename {
	return &Rename{Source: source, Target: target}
}

type Rename struct {
	Source string
	Target string
}

func (this *Rename) Name() string {
	return "重命名"
}

func (this *Rename) Run(ctx context.Context, m *tdx.Manage) error {
	es, err := os.ReadDir(this.Source)
	if err != nil {
		return err
	}
	r := &Range[os.DirEntry]{
		Codes:   es,
		Limit:   1,
		Retry:   3,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *Rename) Handler(ctx context.Context, m *tdx.Manage, e os.DirEntry) error {
	return os.Rename(filepath.Join(this.Source, e.Name()), filepath.Join(this.Target, e.Name()))
}
