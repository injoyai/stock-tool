package main

import (
	"os"
	"strings"

	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
)

func main() {

	codes := map[string]bool{}

	err := oss.RangeFile("./data/from/codes", func(info *oss.FileInfo, f *os.File) (bool, error) {
		logs.Debug(info.Name())
		m, err := excel.FromReader(f)
		if err != nil {
			return false, err
		}
		for _, lls := range m {
			for _, ls := range lls {
				if len(ls) >= 8 {
					codes[a(ls[4])] = true
				}

			}
		}
		return true, nil
	})
	logs.PanicErr(err)

	for code, _ := range codes {
		logs.Debug(code)
	}
	// return

	err = oss.RangeFileInfo("./data/from/csv_", func(info *oss.FileInfo) (bool, error) {
		//logs.Debug(info.Name())
		if codes[info.Name()] {
			logs.Debug(info.FullName(), "./data/export/"+info.Name())
			os.Rename(info.FullName(), "./data/export/"+info.Name())
			//oss.RemoveAll("./data/from/codes/" + info.Name())
		}
		return true, nil
	})
	logs.PanicErr(err)
}

func a(code string) string {
	if len(code) == 6 {
		switch {
		case strings.HasPrefix(code, "0") || strings.HasPrefix(code, "30"):
			return "sz" + code
		case strings.HasPrefix(code, "6"):
			return "sh" + code
		}
	}
	return code
}
