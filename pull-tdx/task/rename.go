package task

import (
	"context"
	"github.com/injoyai/tdx"
	"os"
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
	return os.Rename(this.Source, this.Target)
}
