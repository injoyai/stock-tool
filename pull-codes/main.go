package main

import (
	"fmt"

	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx/extend"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.3.2")
	logs.Info("详情:", "升级tdx版本,增加gbbq")
	fmt.Println("===============================================")
}

var (
	Port = cfg.GetInt("port", 8080)
)

func main() {
	err := extend.ListenCodesAndGbbqHTTP(Port, nil, nil)
	logs.Err(err)
}
