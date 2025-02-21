package main

import (
	"github.com/injoyai/goutil/oss/tray"
	"github.com/injoyai/tdx"
	"time"
)

func main() {

	tray.Run(

		func(s *tray.Tray) {
			var err error
			var c *Client
			for {
				c, err = NewClient()
				if err != nil {
					s.SetHint("连接服务器错误: " + err.Error())
					<-time.After(time.Second * 2)
					continue
				}
				s.SetHint("连接服务器成功")
				break
			}
			_ = c

		},

		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),
	)

}

func NewClient() (*Client, error) {
	m, err := tdx.NewManage(&tdx.ManageConfig{})
	if err != nil {
		return nil, err
	}
	return &Client{
		Manage: m,
	}, nil
}

type Client struct {
	*tdx.Manage
}
