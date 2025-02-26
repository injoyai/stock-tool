package main

import (
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/tdx"
)

var (
	config = &tdx.ManageConfig{
		Hosts:  cfg.GetStrings("hosts"),
		Number: cfg.GetInt("number", 1),
		Dir:    cfg.GetString("dir", "./data"),
	}
	disks = cfg.GetInt("disks", 1)
)

func main() {
	//m, err := tdx.NewManage(config)
}
