package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"strings"
	"time"
)

func bySector(codes []string, start, end time.Time, size int) {

	resp, err := pull(codes, start, end)
	logs.PanicErr(err)

	save := func(data [][]any, name string) error {
		buf, err := excel.ToCsv(data)
		if err != nil {
			return err
		}
		return oss.New("./data/csv/"+name+conv.String(time.Now().UnixMilli())+".csv", buf)
	}

	countSh := 0
	countSz0 := 0
	countSz30 := 0
	dataSh := [][]any{title}
	dataSz0 := [][]any{title}
	dataSz30 := [][]any{title}

	for code, ls := range resp {
		switch {
		case strings.HasPrefix(code, "sh"):
			if countSh >= size {
				//保存
				if err = save(dataSh, "沪市"); err != nil {
					logs.Err(err)
					continue
				}
				dataSh = [][]any{title}
				countSh = 0
			}
			countSh++
			for _, v := range ls {
				dataSh = append(dataSh, body(code, v))
			}
		case strings.HasPrefix(code, "sz0"):
			if countSz0 >= size {
				//保存
				if err = save(dataSz0, "深市"); err != nil {
					logs.Err(err)
					continue
				}
				dataSz0 = [][]any{title}
				countSz0 = 0
			}
			countSz0++
			for _, v := range ls {
				dataSz0 = append(dataSz0, body(code, v))
			}
		case strings.HasPrefix(code, "sz30"):
			if countSz30 >= size {
				//保存
				if err = save(dataSz30, "科创"); err != nil {
					logs.Err(err)
					continue
				}
				dataSz30 = [][]any{title}
				countSz30 = 0
			}
			countSz30++
			for _, v := range ls {
				dataSz30 = append(dataSz30, body(code, v))
			}
		}
	}

	logs.PrintErr(save(dataSh, "沪市"))

	logs.PrintErr(save(dataSz0, "深市"))

	logs.PrintErr(save(dataSz30, "科创"))

}
