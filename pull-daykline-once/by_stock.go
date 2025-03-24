package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"path/filepath"
	"time"
)

func byStock(codes []string, start, end time.Time) {

	resp, err := pull(codes, start, end)
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
