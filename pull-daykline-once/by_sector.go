package main

import (
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"strings"
	"time"
)

func bySector(codes []string, start, end time.Time) {

	resp, err := pull(codes, start, end)
	logs.PanicErr(err)

	dataSh := [][]any{title}
	dataSz0 := [][]any{title}
	dataSz30 := [][]any{title}

	for code, ls := range resp {
		switch {
		case strings.HasPrefix(code, "sh"):
			for _, v := range ls {
				dataSh = append(dataSh, body(code, v))
			}
		case strings.HasPrefix(code, "sz0"):
			for _, v := range ls {
				dataSz0 = append(dataSz0, body(code, v))
			}
		case strings.HasPrefix(code, "sz30"):
			for _, v := range ls {
				dataSz30 = append(dataSz30, body(code, v))
			}
		}
	}

	buf, err := excel.ToCsv(dataSh)
	logs.PanicErr(err)
	oss.New("./data/csv/沪市.csv", buf)

	buf, err = excel.ToCsv(dataSz0)
	logs.PanicErr(err)
	oss.New("./data/csv/深市.csv", buf)

	buf, err = excel.ToCsv(dataSz30)
	logs.PanicErr(err)
	oss.New("./data/csv/科创.csv", buf)

}
