package main

import (
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx/extend"
)

var (
	address = cfg.GetString("address", "http://127.0.0.1:20000")
)

func main() {
	cs, err := extend.DialCodesHTTP(address)
	logs.PanicErr(err)

	codes := cs.GetETFCodes()
	for _, v := range codes {
		logs.Debug(v)
	}

	logs.Debug("总数:", len(codes))

}
