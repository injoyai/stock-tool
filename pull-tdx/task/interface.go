package task

import (
	"context"
	"github.com/injoyai/tdx"
)

type Tasker interface {
	Name() string
	Run(ctx context.Context, m *tdx.Manage) error
}

type Handler[T any] interface {
	Name() string
	Handler(ctx context.Context, m *tdx.Manage, code T) error
}
