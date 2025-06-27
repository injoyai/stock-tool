package main

import (
	"context"
	"github.com/injoyai/base/chans"
)

type HHDOperator[T1, T2 any] interface {
	Read() ([]T1, error) //从硬盘中读取
	Deal(T1) (T2, error) //处理数据
	Write(T2) error      //写入硬盘
}

func NewHHD[T1, T2 any](hhd HHDOperator[T1, T2], dealLimit int, dealErr func(err error), saveErr func(err error) (exit bool)) *HHD[T1, T2] {
	return &HHD[T1, T2]{
		HHDOperator: hhd,
		DealLimit:   dealLimit,
		DealErr:     dealErr,
		SaveErr:     saveErr,
	}
}

type HHD[T1, T2 any] struct {
	HHDOperator[T1, T2]
	DealLimit int
	DealErr   func(err error)
	SaveErr   func(err error) (exit bool)
}

func (this *HHD[T1, T2]) Run(ctx context.Context) error {

	//1. 从硬盘读取数据
	ls, err := this.Read()
	if err != nil {
		return err
	}

	//2. 协程处理数据
	ch := make(chan T2)
	go func(ls []T1) {
		wg := chans.NewWaitLimit(this.DealLimit)
		for _, v := range ls {
			wg.Add()
			go func(v T1) {
				defer wg.Done()
				rs, err := this.Deal(v)
				if err != nil {
					if this.DealErr != nil {
						this.DealErr(err)
					}
					return
				}
				ch <- rs
			}(v)
		}
		wg.Done()
	}(ls)

	//3. 单线程写入硬盘
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			if err := this.Write(data); err != nil {
				if this.SaveErr != nil {
					if this.SaveErr(err) {
						return err
					}
				}
			}
		}
	}

}
