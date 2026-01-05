package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
}

func main() {

	defer func() { g.Input("按回车键退出...") }()

	dirHistory := "./历史"
	dirThisyear := "./当年"
	dir3 := "./合并"

	if !oss.Exists(dirHistory) {
		logs.Err("文件夹[历史]不存在")
		return
	}

	if !oss.Exists(dirThisyear) {
		logs.Err("文件夹[当年]不存在")
		return
	}

	err := os.MkdirAll(dir3, 0777)
	if err != nil {
		logs.Err(err)
		return
	}

	oss.RangeFile(dirHistory, func(info *oss.FileInfo, f *os.File) (bool, error) {

		logs.Info(info.Name())

		filename2025 := filepath.Join(dirThisyear, info.Name())

		output := filepath.Join(dir3, info.Name())
		ff, err := os.Create(output)
		if err != nil {
			logs.Err(err)
			return true, nil
		}
		defer ff.Close()

		bs, err := io.ReadAll(f)
		if err != nil {
			logs.Err(err)
			return true, nil
		}

		ff.Write(bs)

		csv.ImportRange(filename2025, func(i int, line []string) bool {
			if i == 0 {
				i++
				return true
			}
			ff.Write([]byte(strings.Join(line, ",") + "\n"))
			return true
		})

		return true, nil

	})

}
