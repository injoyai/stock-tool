package main

import (
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
}

func main() {

	defer func() { g.Input("按回车键退出...") }()

	dir1 := "./2000-2024"
	dir2 := "./2025"
	dir3 := "./2000-2025"

	if !oss.Exists(dir1) {
		logs.Err("文件夹2000-2024不存在")
		return
	}

	if !oss.Exists(dir2) {
		logs.Err("文件夹2025不存在")
		return
	}

	err := os.MkdirAll(dir3, 0777)
	if err != nil {
		logs.Err(err)
		return
	}

	oss.RangeFile(dir1, func(info *oss.FileInfo, f *os.File) (bool, error) {

		logs.Info(info.Name())

		filename2025 := filepath.Join(dir2, info.Name())

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

		i := 0
		csv.ImportRange(filename2025, func(line []string) bool {
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
