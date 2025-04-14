package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/notice"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/tray"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"os"
	"time"
)

var (
	filename = oss.UserInjoyDir("/monitor-price/config/config.json")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
}

func main() {
	mon := &monitor{}
	tray.Run(
		func(s *tray.Tray) {

			go func() {
				for {
					c, err := tdx.DialDefault()
					if err == nil {
						mon.Client = c
						break
					}
					logs.Err(err)
				}
				bs, _ := os.ReadFile(filename)
				mon.setConfig(bs)
				mon.Run(context.Background())
			}()

		},
		tray.WithHint("监听价格"),
		tray.WithShow(func(m *tray.Menu) { gui() }),
		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),
	)
}

func gui() {
	lorca.Run(&lorca.Config{
		Width:  900,
		Height: 640,
		Index:  "./index.html",
	}, func(app lorca.APP) error {

		app.Bind("getConfig", func() any {
			bs, _ := os.ReadFile(filename)
			m := map[string]any{}
			json.Unmarshal(bs, &m)
			return m
		})

		app.Bind("setConfig", func(cfg any) {
			oss.New(filename, cfg)
		})

		app.Eval("initialize()")
		app.Eval("window.onload = initialize;")

		return nil
	})
}

type monitor struct {
	*tdx.Client
	interval time.Duration
	codes    map[string]Config
}

func (this *monitor) setConfig(cfg any) {
	m := conv.NewMap(cfg)
	this.interval = m.GetSecond("interval")
	if this.interval < time.Second {
		this.interval = time.Second * 10
	}
	this.codes = func() map[string]Config {
		result := make(map[string]Config)
		for _, v := range m.GetInterfaces("rule") {
			m2 := conv.NewMap(v)
			result[m2.GetString("code")] = Config{
				Code:    m2.GetString("code"),
				Price:   protocol.Price(m2.GetFloat64("price") * 1000),
				Greater: m2.GetBool("greater"),
				Enable:  m2.GetBool("enable"),
			}
		}
		return result
	}()
}

func (this *monitor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(this.interval):
			logs.Debug("codes:", this.codes)
			for code, config := range this.codes {
				if !config.Enable {
					continue
				}
				resp, err := this.Client.GetMinuteTrade(code, 0, 1)
				if err != nil {
					logs.Err(err)
					continue
				}
				if len(resp.List) > 0 {
					logs.Info(code, resp.List[0].Price)
					lastPrice := resp.List[0].Price
					if config.Greater && lastPrice <= config.Price {
						notice.DefaultWindows.Publish(&notice.Message{
							Content: fmt.Sprintf("代码[%s],[%.2f]大于阈值[%.2f]", code, lastPrice.Float64(), config.Price.Float64()),
						})
					} else if !config.Greater && resp.List[0].Price >= config.Price {
						notice.DefaultWindows.Publish(&notice.Message{
							Content: fmt.Sprintf("代码[%s],[%.2f]小于阈值[%.2f]", code, lastPrice.Float64(), config.Price.Float64()),
						})
					}

				}
			}
		}
	}
}

type Config struct {
	Code    string
	Price   protocol.Price
	Greater bool
	Enable  bool
}
