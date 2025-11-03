package main

import (
	"fmt"
	"github.com/injoyai/frame/fiber"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.1")
	logs.Info("详情:", "初版")
	fmt.Println("===============================================")
}

func main() {
	filename := filepath.Join(tdx.DefaultDatabaseDir, "codes.db")
	cs, err := DialCodes(filename)
	logs.PanicErr(err)
	s := fiber.Default()
	s.Group("/api", func(g fiber.Grouper) {
		g.ALL("/stocks", func(c fiber.Ctx) { c.Succ(cs.GetStocks()) })
		g.ALL("/etfs", func(c fiber.Ctx) { c.Succ(cs.GetEtfs()) })
	})
	logs.Err(s.Run())
}
