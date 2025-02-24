package main

import (
	"context"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss/tray"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
)

func main() {

	tray.Run(

		func(s *tray.Tray) {
			var err error
			var c *Client
			for {
				c, err = NewClient(nil, 100)
				if err != nil {
					s.SetHint("连接服务器错误: " + err.Error())
					<-time.After(time.Second * 2)
					continue
				}
				s.SetHint("连接服务器成功")
				break
			}

			//收盘后开始更新数据
			c.AddWorkdayTask("0 1 15 * * *", func(m *tdx.Manage) {
				c.Update(context.Background())

			})

		},

		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),
	)

}

func NewClient(cfg *tdx.ManageConfig, limit int) (*Client, error) {
	m, err := tdx.NewManage(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		Manage: m,
	}, nil
}

type Client struct {
	*tdx.Manage
	dir  string
	read chan interface{}
	save chan interface{}
}

func (this *Client) Update(ctx context.Context) {
	codes, err := this.Codes.Code(true)
	if err != nil {

		return
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, v := range codes {
					db, err := sqlite.NewXorm(filepath.Join(this.dir, v.Code+".db"))
					if err != nil {
						logs.Err(err)
						continue
					}
					this.read <- db
				}
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		for i := range codes {
			select {
			case <-ctx.Done():
			default:
				code := codes[i]
				this.Go(func(c *tdx.Client) {
					c.GetKlineDayUntil(code.Code, func(k *protocol.Kline) bool {

						return true
					})
				})
			}
		}
	}(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for _, v := range codes {

			select {
			case <-ctx.Done():
				return
			default:
			}

			db, err := sqlite.NewXorm(filepath.Join(this.dir, v.Code+".db"))
			if err != nil {
				logs.Err(err)
				continue
			}
			this.read <- db
		}

	}

}
