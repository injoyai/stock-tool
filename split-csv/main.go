package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
)

func main() {
	split("1分钟")
	split("5分钟")
	split("15分钟")
	split("30分钟")
	split("60分钟")
}

func split(dir string) error {
	es, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	b := bar.New(
		bar.WithTotal(int64(len(es))),
		bar.WithPrefix("["+dir+"][xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()
	for _, v := range es {
		err = deal(filepath.Join(dir, v.Name()), filepath.Join("export", dir), v.Name())
		if err != nil {
			b.Logf("[错误] %s", err)
		}
		b.Add(1)
		b.Flush()
	}
	return nil
}

func deal(filename, exportDir, name string) error {
	title := []string(nil)
	m := map[int][][]string{}
	x := map[int]map[string]struct{}{}
	err := csv.ImportRange(filename, func(i int, line []string) bool {
		if i == 0 {
			title = line
			return true
		}
		if len(line) == 0 {
			return true
		}
		t, err := time.Parse(time.DateTime, line[0])
		if err != nil {
			logs.Err(err)
			return false
		}

		year := t.Year()

		xx := x[year]
		if xx == nil {
			xx = map[string]struct{}{}
			x[year] = xx
		}

		if _, ok := xx[line[0]]; ok {
			return true
		}

		xx[line[0]] = struct{}{}

		m[year] = append(m[year], line)
		return true
	})
	if err != nil {
		return err
	}

	for year, lss := range m {
		data := [][]any{conv.Interfaces(title)}
		for _, v := range lss {
			data = append(data, conv.Interfaces(v))
		}
		buf, err := csv.Export(data)
		if err != nil {
			return err
		}
		fn := filepath.Join(exportDir, conv.String(year), name)
		err = oss.New(fn, buf)
		if err != nil {
			return err
		}
	}

	return nil
}
