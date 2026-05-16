package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/regexps"
	"github.com/injoyai/logs"
)

func init() {
	//logs.SetFormatter(logs.TimeFormatter)
}

func main() {

	defer func() { g.Input("按回车键退出...") }()

	es, err := os.ReadDir("./")
	logs.PanicErr(err)

	dirs := make([]string, 0)
	for _, e := range es {
		if len(e.Name()) >= 4 && regexps.Is("^[0-9]{4}.*?", e.Name()) {
			dirs = append(dirs, e.Name())
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i] < dirs[j]
	})

	if len(dirs) <= 1 {
		logs.Err("文件夹不足")
		return
	}

	outputDir := dirs[0][:4] + "-" + dirs[len(dirs)-1][:4]

	err = do(dirs, outputDir, "1分钟")
	logs.PrintErr(err)

	err = do(dirs, outputDir, "5分钟")
	logs.PrintErr(err)

}

func do(dirs []string, outputDir, _type string) error {
	historyDir := filepath.Join(dirs[0], _type)

	es, err := os.ReadDir(historyDir)
	if err != nil {
		return err
	}

	b := bar.New(
		bar.WithTotal(int64(len(es))),
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for _, e := range es {

		err := func() error {
			b.SetPrefix("[" + e.Name() + "]")
			b.Flush()

			defer func() {
				b.Add(1)
				b.Flush()
			}()

			err = os.MkdirAll(filepath.Join(outputDir, _type), 0777)
			if err != nil {
				return err
			}
			outputFilename := filepath.Join(outputDir, _type, e.Name())
			out, err := os.Create(outputFilename)
			if err != nil {
				logs.Err(err)
				return err
			}
			defer out.Close()

			historyFilename := filepath.Join(historyDir, e.Name())
			historyFile, err := os.Open(historyFilename)
			if err != nil {
				return err
			}
			defer historyFile.Close()

			if _, err = io.Copy(out, historyFile); err != nil {
				return err
			}

			for _, dir := range dirs[1:] {
				if dir == outputDir {
					continue
				}
				filename := filepath.Join(dir, _type, e.Name())
				err = csv.ImportRange(filename, func(i int, line []string) bool {
					if i == 0 {
						i++
						return true
					}
					out.Write([]byte(strings.Join(line, ",") + "\n"))
					return true
				})
				if err != nil {
					return err
				}
			}
			return nil
		}()
		if err != nil {
			b.Logf("[错误][%s] %v", e, err)
			b.Flush()
			continue
		}
	}

	return nil
}
