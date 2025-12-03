package main

import (
	"fmt"

	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/frame/fiber"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.3.1")
	logs.Info("详情:", "增加指数代码")
	fmt.Println("===============================================")
}

var (
	Database = cfg.GetString("database", "./data/database/codes.db")
	Spec     = cfg.GetString("spec", "0 10 9 * * *")
	Port     = cfg.GetInt("port", 8080)
)

func main() {

	cs, err := tdx.NewCodes2(
		tdx.WithDBFilename(Database),
		tdx.WithSpec(Spec),
	)
	logs.PanicErr(err)

	s := fiber.Default()
	s.SetPort(Port)
	s.Group("/api", func(g fiber.Grouper) {
		g.ALL("/stocks", func(c fiber.Ctx) { data := cs.GetStocks(); c.Succ(data, int64(len(data))) })
		g.ALL("/etfs", func(c fiber.Ctx) { data := cs.GetETFs(); c.Succ(data, int64(len(data))) })
		g.ALL("/indexes", func(c fiber.Ctx) { data := cs.GetIndexes(); c.Succ(data, int64(len(data))) })
	})
	logs.Err(s.Run())
}
