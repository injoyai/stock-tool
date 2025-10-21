package main

import (
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
)

var (
	Indexes = cfg.GetStrings("indexes")
	Clients = cfg.GetInt("clients")
	Codes   = cfg.GetStrings("codes")
)

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStocks()
	}

	Codes = append(Codes, Indexes...)

	cr := cron.New(cron.WithSeconds())

	cr.AddFunc("0 20 15 * * *", func() {
		
	})

}

func pull(m *tdx.Manage, codes []string) error {

}
