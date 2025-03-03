package task

import (
	"context"
	"github.com/injoyai/base/maps"
	"sync"
)

type Tasker interface {
	Name() string
	Run(ctx context.Context) error
	//Running() bool
	//RunInfo() string
}

func New() *Manage {
	m := &Manage{}
	go m.run(context.Background())
	return m
}

type Manage struct {
	ch chan Tasker
	m  *maps.Safe
	mu sync.Mutex
}

func (this *Manage) Names() []string {
	names := []string(nil)
	this.m.Range(func(key, value interface{}) bool {
		names = append(names, value.(Tasker).Name())
		return true
	})
	return names
}

func (this *Manage) Add(t Tasker) {
	this.ch <- t
	this.m.Set(t, t)
}

func (this *Manage) run(ctx context.Context) error {
	for v := range this.ch {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			v.Run(ctx)
			this.m.Del(v)
		}
	}
	return nil
}
