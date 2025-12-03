package main

import (
	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
)

var (
	clients    = cfg.GetInt("clients", 3)
	coroutines = cfg.GetInt("coroutines", 10)
	retry      = cfg.GetInt("retry", 3)
	address    = cfg.GetString("address", "http://127.0.0.1:20000")
	spec       = cfg.GetString("spec", "0 15 15 * * *")
)

func main() {

	m, err := tdx.NewManage(
		tdx.WithClients(clients),
		tdx.WithDialCodes(func(c *tdx.Client, database string) (tdx.ICodes, error) {
			return DialCodesHTTP(address)
		}),
	)
	logs.PanicErr(err)

	//更新
	logs.PanicErr(Update(m))

	m.AddWorkdayTask(spec, func(m *tdx.Manage) {
		logs.PrintErr(Update(m))
	})

	select {}
}

func Update(m *tdx.Manage) error {
	codes := m.Codes.GetETFCodes()

	b := bar.NewCoroutine(len(codes), coroutines)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.GoRetry(func() error {
			return m.Do(func(c *tdx.Client) error { return update(c, code) })
		}, retry)
	}

	return nil
}

func update(c *tdx.Client, code string) error {

	//读取数据库最后一条数据

	//拉取数据

	//更新到数据库

	return nil
}

// exportThisYear 导出今年数据
func exportThisYear() error {

	//打开数据库

	//读取今年数据

	//导出

	return nil
}
