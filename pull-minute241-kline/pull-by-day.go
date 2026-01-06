package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

func PullByDay(m *tdx.Manage, date time.Time, codes []string, goroutines int, exportDir, uploadDir string) error {
	if times.IntegerDay(date) == times.IntegerDay(time.Now()) {
		if date.Hour() < 15 {
			return errors.New("需要在15点之后更新")
		}
	}

	os.MkdirAll(exportDir, 0755)
	os.MkdirAll(uploadDir, 0755)

	name := date.Format(time.DateOnly)

	b := bar.NewCoroutine(len(codes), goroutines, bar.WithPrefix("[增量]"))
	defer b.Close()

	for _, code := range codes {
		b.Go(func() {
			err := g.Retry(func() error {
				return m.Do(func(c *tdx.Client) error {
					return pullCode(m.Gbbq, c, code, date, filepath.Join(exportDir, name))
				})
			}, DefaultRetry, time.Second*2)
			logs.PrintErr(err)
		})
	}
	b.Wait()

	logs.Debug("压缩...")
	err := zip.Encode(filepath.Join(exportDir, name), filepath.Join(exportDir, name+".zip"))
	logs.PrintErr(err)

	logs.Debug("重命名...")
	return os.Rename(filepath.Join(exportDir, name+".zip"), filepath.Join(uploadDir, name+".zip"))

}

func pullCode(gb tdx.IGbbq, c *tdx.Client, code string, date time.Time, exportDir string) error {
	start := times.IntegerDay(date)
	end := start.AddDate(0, 0, 1).Add(-1)
	resp, err := c.GetKlineMinute241Until(code, func(k *protocol.Kline) bool {
		return k.Time.Before(start)
	})
	if err != nil {
		return err
	}
	ks := protocol.Klines{}
	for _, v := range resp.List {
		if v.Time.Before(start) {
			continue
		}
		if v.Time.After(end) {
			continue
		}
		ks = append(ks, v)
	}

	if err = save(gb, code, ks, filepath.Join(exportDir, "1分钟")); err != nil {
		return err
	}

	if err = save(gb, code, ks.Merge241(5), filepath.Join(exportDir, "5分钟")); err != nil {
		return err
	}

	if err = save(gb, code, ks.Merge241(15), filepath.Join(exportDir, "15分钟")); err != nil {
		return err
	}

	if err = save(gb, code, ks.Merge241(30), filepath.Join(exportDir, "30分钟")); err != nil {
		return err
	}

	if err = save(gb, code, ks.Merge241(60), filepath.Join(exportDir, "60分钟")); err != nil {
		return err
	}

	return nil
}

func save(gb tdx.IGbbq, code string, ks protocol.Klines, exportDir string) error {
	data := [][]any{Title}
	for _, v := range ks {
		x := []any{
			code,
			v.Time.Format(time.DateTime),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume * 100,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
			gb.GetTurnover(code, v.Time, v.Volume*100),
		}
		if eq := gb.GetEquity(code, v.Time); eq != nil {
			x = append(x, int64(eq.Float), int64(eq.Total))
		}
		data = append(data, x)
	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	err = oss.New(filepath.Join(exportDir, code+".csv"), buf)
	if err != nil {
		return err
	}

	return nil
}
