package main

import (
	"errors"
	"fmt"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/robfig/cron/v3"
	"path/filepath"
	"time"
)

var (
	Dir           = "./data"
	Retry         = 3
	RetryInterval = time.Millisecond * 200
	Spec          = cfg.GetString("spec", "0 31 9 * * *")
	Startup       = cfg.GetBool("startup", false)
)

func main() {
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)
	c := cron.New(cron.WithSeconds())
	c.AddFunc(Spec, func() { logs.PrintErr(Pull(m)) })
	if Startup {
		logs.PrintErr(Pull(m))
	}
	c.Run()
}

func Pull(m *tdx.Manage) error {
	if !m.Workday.Is(time.Now().AddDate(0, 0, -1)) {
		return errors.New("昨天不是工作日")
	}

	codes := m.Codes.GetStocks()

	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for _, v := range m.Codes.GetStocks() {
		b.SetPrefix(fmt.Sprintf("[%s]", v))
		b.Flush()
		err := m.Do(func(c *tdx.Client) error {
			return g.Retry(func() error { return pull(c, v, "复权因子") }, Retry, RetryInterval)
		})
		b.Add(1)
		b.Flush()
		if err != nil {
			b.Logf("[错误] %s", err)
			b.Flush()
		}
	}

	return zip.Encode(filepath.Join(Dir, "复权因子"), filepath.Join(Dir, "upload", "复权因子.zip"))
}

func pull(c *tdx.Client, code string, folder string) error {
	_, fs, err := extend.GetTHSDayKlineFactorFull(code, c)
	if err != nil {
		return err
	}
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	data := [][]any{{"日期", "前复权因子", "后复权因子"}}
	for _, v := range fs {
		if v.Date >= end.Unix() {
			break
		}
		data = append(data, []any{
			time.Unix(v.Date, 0).Format("2006-01-02"),
			v.QFactor,
			v.HFactor,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(Dir, folder, code+".csv")
	return oss.New(filename, buf)
}
