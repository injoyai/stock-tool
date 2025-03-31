package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

func main() {
	defer done()()

	byStock(nil, time.Time{}, time.Now().Add(time.Hour*24), func(c *tdx.Client) Handler { return c.GetKline15MinuteUntil })

}

func byStock(codes []string, start, end time.Time, f func(c *tdx.Client) Handler) {

	resp, err := pull(codes, start, end, f)
	logs.PanicErr(err)

	for code, ls := range resp {
		data := [][]any{title}
		for _, v := range ls {
			data = append(data, body(code, v))
		}
		buf, err := excel.ToCsv(data)
		logs.PanicErr(err)
		oss.New(filepath.Join("./data/csv", code+".csv"), buf)
	}

}
