package api

import (
	"errors"
	"github.com/injoyai/base/safe"
	"github.com/injoyai/tdx"
)

func NewPool(dial func() (*tdx.Client, error), cap int) (*Pool, error) {
	ch := make(chan *tdx.Client, cap)
	p := &Pool{
		ch: ch,
		Closer: safe.NewCloser().SetCloseFunc(func(err error) error {
			close(ch)
			return nil
		}),
	}
	for i := 0; i < cap; i++ {
		c, err := dial()
		if err != nil {
			return nil, err
		}
		p.ch <- c
	}
	return p, nil
}

type Pool struct {
	ch chan *tdx.Client
	*safe.Closer
}

func (this *Pool) Get() (*tdx.Client, error) {
	select {
	case <-this.Done():
		return nil, this.Err()
	case c, ok := <-this.ch:
		if !ok {
			return nil, errors.New("已关闭")
		}
		return c, nil
	}
}

func (this *Pool) Put(c *tdx.Client) {
	select {
	case <-this.Done():
		c.Close()
		return
	case this.ch <- c:
	}
}

func (this *Pool) Do(fn func(c *tdx.Client) error) error {
	c, err := this.Get()
	if err != nil {
		return err
	}
	defer this.Put(c)
	return fn(c)
}

func (this *Pool) Go(fn func(c *tdx.Client)) error {
	c, err := this.Get()
	if err != nil {
		return err
	}
	go func(c *tdx.Client) {
		defer this.Put(c)
		fn(c)
	}(c)
	return nil
}
