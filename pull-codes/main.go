package main

import (
	"fmt"
	"github.com/injoyai/frame/fiber"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2")
	logs.Info("详情:", "增加流通/总股本")
	fmt.Println("===============================================")
}

func main() {

	cs, err := tdx.NewCodes2(tdx.WithDBFilename("./data/database/codes.db"))
	logs.PanicErr(err)

	s := fiber.Default()
	s.Group("/api", func(g fiber.Grouper) {
		g.ALL("/stocks", func(c fiber.Ctx) { c.Succ(cs.GetStocks()) })
		g.ALL("/etfs", func(c fiber.Ctx) { c.Succ(cs.GetETFs()) })
	})
	logs.Err(s.Run())
}
