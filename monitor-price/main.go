package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/notice"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/tray"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/lorca"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"os"
	"time"
)

//go:embed index.html
var index string

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
				codes, _ := tdx.NewCodes(mon.Client, oss.UserInjoyDir("/monitor-price/codes.db"))
				mon.getName = func(code string) string {
					if codes == nil {
						return code
					}
					return codes.GetName(code)
				}
				bs, _ := os.ReadFile(filename)
				mon.setConfig(bs)
				mon.Run(context.Background(), s)
			}()

		},
		tray.WithIco(Ico),
		tray.WithHint("监听价格"),
		tray.WithShow(func(m *tray.Menu) { gui(mon) }),
		tray.WithButton("刷新", func(m *tray.Menu) { mon.Refresh() }),
		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),
	)
}

func gui(mon *monitor) {
	lorca.Run(&lorca.Config{
		Width:  900,
		Height: 640,
		Index:  index,
	}, func(app lorca.APP) error {

		app.Bind("getConfig", func() any {
			bs, _ := os.ReadFile(filename)
			m := map[string]any{}
			json.Unmarshal(bs, &m)
			return m
		})

		app.Bind("setConfig", func(cfg any) {
			mon.setConfig(cfg)
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
	getName  func(code string) string
	hand     chan struct{}
	refresh  bool
}

func (this *monitor) Refresh() {
	this.refresh = true
}

func (this *monitor) setConfig(cfg any) {
	m := conv.NewMap(cfg)
	this.interval = m.GetSecond("interval")
	if this.interval < time.Second {
		this.interval = time.Second * 10
	}
	this.refresh = true
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

func (this *monitor) Run(ctx context.Context, s *tray.Tray) error {
	interval := time.Duration(0)

	f := func() {
		if this.Client == nil {
			return
		}
		now := time.Now()
		hint := fmt.Sprintf("数据时间: %s", now.Format(time.TimeOnly))
		defer func() {
			this.refresh = false
			s.SetHint(hint)
		}()
		if !this.refresh {
			if now.Before(times.IntegerDay(now).Add(time.Hour*9 + time.Minute*30)) {
				return
			}
			if now.After(times.IntegerDay(now).Add(time.Hour * 15)) {
				return
			}
			if now.After(times.IntegerDay(now).Add(time.Hour*11+time.Minute*30)) &&
				now.Before(times.IntegerDay(now).Add(time.Hour*13)) {
				return
			}
		}
		for code, config := range this.codes {
			if !config.Enable {
				continue
			}
			resp, err := this.Client.GetKlineDay(code, 0, 1)
			if err != nil {
				logs.Err(err)
				continue
			}
			if len(resp.List) > 0 {
				lastPrice := resp.List[0].Close
				info := fmt.Sprintf("%s: %.2f", this.getName(code), lastPrice.Float64())
				hint += "\n" + info
				logs.Info(info, "  大于阈值:", lastPrice >= config.Price)
				if config.Greater && lastPrice >= config.Price {
					if config.limit < 0 {
						//向上突破阈值,发送通知
						notice.DefaultWindows.Publish(&notice.Message{
							Content: fmt.Sprintf("代码[%s],[%.2f]大于阈值[%.2f]", this.getName(code), lastPrice.Float64(), config.Price.Float64()),
						})
					}
					config.limit = 1

				} else if !config.Greater && lastPrice <= config.Price {
					if config.limit > 0 {
						//向下突破阈值,发送通知
						notice.DefaultWindows.Publish(&notice.Message{
							Content: fmt.Sprintf("代码[%s],[%.2f]小于阈值[%.2f]", this.getName(code), lastPrice.Float64(), config.Price.Float64()),
						})
					}
					config.limit = -1

				}

			}
		}
	}

	for i := 0; ; i++ {
		if i > 0 {
			interval = this.interval
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-this.hand:
			f()
		case <-time.After(interval):
			f()
		}
	}
}

type Config struct {
	Code    string
	Price   protocol.Price
	Greater bool
	Enable  bool
	limit   int8 //相对阈值状态 -1(阈值下),0,1(阈值上)
}

/*



 */

var Ico = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x20, 0x20, 0x00, 0x00, 0x01, 0x00,
	0x20, 0x00, 0xA8, 0x10, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00,
	0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xDB, 0xA7, 0x48, 0x40, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xAA,
	0x48, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xAA, 0x49, 0xBF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xD7, 0xA7, 0x48, 0x20, 0xDB, 0xA7, 0x48, 0x40, 0xDA, 0xA9,
	0x49, 0xFF, 0xDB, 0xA9, 0x48, 0x7F, 0xD7, 0xA7, 0x48, 0x20, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0xD7, 0xA7, 0x48, 0x20, 0xD7, 0xA7,
	0x48, 0x20, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xDF, 0xD7, 0xA7,
	0x48, 0x20, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0xDA, 0xAA, 0x48, 0x60, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xAA,
	0x48, 0x60, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA7, 0x48, 0x40, 0xDA, 0xA9,
	0x49, 0xDF, 0xD7, 0xA7, 0x48, 0x20, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xAA,
	0x49, 0xBF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDB, 0xA9, 0x48, 0x7F, 0xD7, 0xA7, 0x48, 0x20, 0xDB, 0xA9,
	0x48, 0x7F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xD7, 0xA7,
	0x48, 0x20, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0xD7, 0xA7,
	0x48, 0x20, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xDF, 0xD7, 0xA7,
	0x48, 0x20, 0x00, 0x00, 0x00, 0x00, 0xD7, 0xA7, 0x48, 0x20, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xAA, 0x48, 0x60, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xAA, 0x49, 0xBF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xDF, 0xD7, 0xA7, 0x48, 0x20, 0xD7, 0xA7, 0x48, 0x20, 0x00, 0x00,
	0x00, 0x00, 0xDA, 0xAA, 0x49, 0xBF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0xDB, 0xA7, 0x48, 0x40, 0x00, 0x00, 0x00, 0x00, 0xD7, 0xA7,
	0x48, 0x20, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDB, 0xA9,
	0x48, 0x7F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xAA, 0x48, 0x60, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xAA, 0x49, 0xBF, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDB, 0xA7, 0x48, 0x40, 0xD7, 0xA7, 0x48, 0x20, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDB, 0xA9,
	0x48, 0x7F, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0x00, 0x00,
	0x00, 0x00, 0xDB, 0xA9, 0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xD7, 0xA7, 0x48, 0x20, 0xDA, 0xA9,
	0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xAA, 0x49, 0xBF, 0xDA, 0xAA,
	0x49, 0xBF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xDF, 0xD7, 0xA7, 0x48, 0x20, 0x00, 0x00, 0x00, 0x00, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xDF, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xAA, 0x48, 0x60, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8, 0x4A, 0x9F, 0xDB, 0xA9,
	0x48, 0x7F, 0xDA, 0xA9, 0x49, 0xDF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA8,
	0x4A, 0x9F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDA, 0xA9,
	0x49, 0xFF, 0xDA, 0xA9, 0x49, 0xFF, 0xDB, 0xA9, 0x48, 0x7F, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xDA, 0xAA, 0x48, 0x60, 0xDA, 0xA8,
	0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA8,
	0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA8,
	0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xA8, 0x4A, 0x9F, 0xDA, 0xAA,
	0x48, 0x60, 0xD7, 0xA7, 0x48, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xF1, 0xFF, 0xFF, 0xFF, 0xE0,
	0xFF, 0xFF, 0xFF, 0xC0, 0x7F, 0xFF, 0xFF, 0x80, 0x3F, 0xFF, 0xFF, 0x00,
	0x1F, 0xFF, 0xFE, 0x00, 0x0F, 0xFF, 0xFC, 0x00, 0x07, 0xFF, 0xF8, 0x00,
	0x03, 0xFF, 0xF0, 0x10, 0x01, 0xFF, 0xE0, 0x08, 0x00, 0xFF, 0xE0, 0x06,
	0x00, 0x7F, 0xE0, 0x02, 0xC0, 0x3F, 0xF0, 0x02, 0x10, 0x1F, 0xF8, 0x02,
	0x00, 0x1F, 0xFC, 0x00, 0x00, 0x1F, 0xFE, 0x01, 0x00, 0x1F, 0xFF, 0x01,
	0x00, 0x1F, 0xFF, 0x80, 0x00, 0x1F, 0xFF, 0xC0, 0x00, 0x1F, 0xFF, 0xE0,
	0x01, 0x1F, 0xFF, 0xF0, 0x03, 0x1F, 0xFF, 0xF8, 0x00, 0x1F, 0xFF, 0xFC,
	0x00, 0x1F, 0xFF, 0xFE, 0x00, 0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF,
}
