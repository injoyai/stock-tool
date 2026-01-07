package main

import (
	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/robfig/cron/v3"
)

var (
	DatabaseDir = cfg.GetString("database_dir", "./data/database/day-kline")
	Clients     = cfg.GetInt("clients", 1)
	Address     = cfg.GetString("address", "http://192.168.192.5:20000")
	Goroutines  = cfg.GetInt("goroutines", 10)
	Codes       = cfg.GetStrings("codes")
	Spec        = cfg.GetString("spec", "0 1 15 * * *")
	Startup     = cfg.GetBool("startup")
)

func main() {

	m, err := tdx.NewManage(
		tdx.WithClients(Clients),
		tdx.WithCodes(nil),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) {
			return extend.DialCodesHTTP(Address)
		}),
		tdx.WithGbbq(nil),
		tdx.WithDialGbbq(func(c *tdx.Client) (tdx.IGbbq, error) {
			return extend.DialGbbqHTTP(Address)
		}),
	)
	logs.PanicErr(err)

	cr := cron.New(cron.WithSeconds())

	cr.AddFunc(Spec, func() {
		logs.PrintErr(run(m, Codes))
	})

	if Startup {
		logs.PrintErr(run(m, Codes))
	}

	cr.Run()

}

func run(m *tdx.Manage, codes []string) error {

	if len(codes) == 0 {
		codes = m.Codes.GetStockCodes()
	}

	b := bar.NewCoroutine(len(codes), Goroutines, bar.WithPrefix("[xx000000]"))
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.Go(func() {
			err := g.Retry(func() error {
				return m.Do(func(c *tdx.Client) error {
					return update(c, code)
				})
			}, tdx.DefaultRetry)
			if err != nil {
				b.Logf("[错误] [%s] %s\n", code, err)
				b.Flush()
			}
		})
	}

	b.Wait()

	return nil
}

func update(c *tdx.Client, code string) error {

	return nil
}

func pull() {

}

func export() {

}
