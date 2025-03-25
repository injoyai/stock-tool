package main

import (
	"github.com/injoyai/conv"
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

	size := 200
	cache := [][]any(nil)

	for i := 0; ; i += size {
		if i >= len(dataSh) {
			break
		}
		switch {
		case i+size >= len(dataSh):
			cache = dataSh[i:]
		default:
			cache = dataSh[i : i+size]
		}
		buf, err := excel.ToCsv(cache)
		logs.PanicErr(err)
		oss.New("./data/csv/沪市"+conv.String(i/size+1)+".csv", buf)
	}

	for i := 0; ; i += size {
		if i >= len(dataSz0) {
			break
		}
		switch {
		case i+size >= len(dataSz0):
			cache = dataSz0[i:]
		default:
			cache = dataSz0[i : i+size]
		}
		buf, err := excel.ToCsv(cache)
		logs.PanicErr(err)
		oss.New("./data/csv/深市"+conv.String(i/size+1)+".csv", buf)
	}

	for i := 0; ; i += size {
		if i >= len(dataSz30) {
			break
		}
		switch {
		case i+size >= len(dataSz30):
			cache = dataSz30[i:]
		default:
			cache = dataSz30[i : i+size]
		}
		buf, err := excel.ToCsv(cache)
		logs.PanicErr(err)
		oss.New("./data/csv/科创"+conv.String(i/size+1)+".csv", buf)
	}

}
