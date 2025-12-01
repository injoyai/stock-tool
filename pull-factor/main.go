package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/robfig/cron/v3"
)

var (
	Dir           = "./data"
	folderTHS     = "复权因子(同花顺)"
	folderSina    = "复权因子(新浪)"
	Retry         = 3
	RetryInterval = time.Millisecond * 200
	SpecTHS       = cfg.GetString("spec_ths", "0 31 9 * * *")
	SpecSina      = cfg.GetString("spec_sina", "0 01 15 * * *")
	Startup       = cfg.GetBool("startup", false)
)

func main() {
	m, err := tdx.NewManage()
	logs.PanicErr(err)
	c := cron.New(cron.WithSeconds())
	c.AddFunc(SpecTHS, func() { logs.PrintErr(PullTHS(m)) })
	c.AddFunc(SpecSina, func() { logs.PrintErr(PullSina(m)) })
	if Startup {
		logs.PrintErr(PullTHS(m))
		logs.PrintErr(PullSina(m))
	}
	c.Run()
}

func PullTHS(m *tdx.Manage) error {
	if !m.Workday.Is(time.Now().AddDate(0, 0, -1)) {
		return errors.New("昨天不是工作日")
	}

	defer func() {
		logs.Info("[完成]", folderTHS)
	}()

	codes := m.Codes.GetStockCodes()
	exportDir := filepath.Join(Dir, folderTHS)

	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for _, v := range codes {
		b.SetPrefix(fmt.Sprintf("[%s]", v))
		b.Flush()
		err := m.Do(func(c *tdx.Client) error {
			return g.Retry(func() error { return pullTHS(c, v, exportDir) }, Retry, RetryInterval)
		})
		b.Add(1)
		b.Flush()
		if err != nil {
			b.Logf("[错误] %s", err)
			b.Flush()
		}
	}

	return zip.Encode(exportDir, filepath.Join(Dir, "upload", folderTHS+".zip"))
}

func PullSina(m *tdx.Manage) error {
	if !m.Workday.Is(time.Now()) {
		return errors.New("今天不是工作日")
	}

	defer func() {
		logs.Info("[完成]", folderSina)
	}()

	codes := m.Codes.GetStockCodes()
	exportDir := filepath.Join(Dir, folderSina)

	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for _, v := range codes {
		b.SetPrefix(fmt.Sprintf("[%s]", v))
		b.Flush()
		retry(func() error { return pullSina(v, exportDir) }, b)
		b.Add(1)
		b.Flush()
	}

	return zip.Encode(exportDir, filepath.Join(Dir, "upload", folderSina+".zip"))
}

func retry(f func() error, b *bar.Bar) {
	err := g.Retry(f, Retry, RetryInterval)
	if err != nil {
		b.Logf("[错误] %s", err)
		b.Flush()
	}
}

func pullTHS(c *tdx.Client, code string, dir string) error {
	_, fs, err := extend.GetTHSDayKlineFactorFull(code, c)
	if err != nil {
		return err
	}
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	data := [][]any{title}
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
	filename := filepath.Join(dir, code+".csv")
	return oss.New(filename, buf)
}

func pullSina(code string, dir string) error {
	fs, err := extend.GetSinaFactorFull(code)
	if err != nil {
		return err
	}
	data := [][]any{title}
	for _, f := range fs {
		data = append(data, []any{
			time.Unix(f.Date, 0).Format("2006-01-02"),
			f.QFactor,
			f.HFactor,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(dir, code+".csv")
	return oss.New(filename, buf)
}

/*






 */

var (
	title = []any{"日期", "前复权因子", "后复权因子"}
)
