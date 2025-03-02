package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
)

func main() {
	c, err := tdx.DialHosts(nil)
	logs.PanicErr(err)
	resp, err := c.GetKlineDayAll("sh000001")
	logs.PanicErr(err)
	//for _, v := range resp.List {
	//	logs.Debug(v)
	//}
	logs.Debug(resp.List[0])
	logs.Debug(resp.Count)
}
