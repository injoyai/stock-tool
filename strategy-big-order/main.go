package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
)

var (
	Boundary   = [2]protocol.Price{100 * 1e7, 10 * 1e7}
	StockLimit = -1
	Codes      = []string{"sz000665"}
)

func main() {

	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStocks(StockLimit)
	}

	NewByCode(
		"sz000665",
		time.Now().AddDate(0, 0, -100),
		time.Now(),
	).Run(m)

}
