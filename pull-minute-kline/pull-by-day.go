package main

import (
	"errors"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/types"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
)

func NewPullByDay(codes []string, coroutines int, export string) *PullByDay {
	return &PullByDay{
		Export:     export,
		Coroutines: coroutines,
		Codes:      codes,
	}
}

type PullByDay struct {
	Export     string
	Coroutines int
	Codes      []string
}

func (this *PullByDay) Run(m *tdx.Manage, date time.Time) error {
	if times.IntegerDay(date) == times.IntegerDay(time.Now()) {
		if date.Hour() < 15 {
			return errors.New("需要在15点之后更新")
		}
	}
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	limit := chans.NewWaitLimit(100)
	b := bar.New()
	b.SetTotal(int64(len(codes)))
	for _, code := range codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			defer func() {
				b.Add(1)
				b.Flush()
			}()
			err := g.Retry(func() error {
				return this.pullCodes(code, date, m)
			}, 3, time.Second*2)
			logs.PrintErr(err)
		}(code)
	}
	limit.Wait()

	oss.RangeFileInfo(this.Export, func(info *oss.FileInfo) (bool, error) {
		if info.IsDir() {
			logs.Debug("压缩:", info.FullName())
			err := zip.Encode(info.FullName(), info.FullName()+".zip")
			logs.PrintErr(err)
		}
		return true, nil
	})

	return nil
}

func (this *PullByDay) pullCodes(code string, date time.Time, m *tdx.Manage) error {
	err := this.pullCode(code, date, "1min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKlineMinute(code, 0, 800)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, date, "5min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline5Minute(code, 0, 800)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, date, "15min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline15Minute(code, 0, 800)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, date, "30min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline30Minute(code, 0, 800)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, date, "60min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKlineHour(code, 0, 800)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	return nil
}

func (this *PullByDay) pullCode(code string, date time.Time, suffix string, fn func(code string) (*protocol.KlineResp, error)) error {
	start := times.IntegerDay(date)
	end := start.AddDate(0, 0, 1).Add(-1)
	resp, err := fn(code)
	if err != nil {
		return err
	}
	mKlines := types.SortMap[string, []*protocol.Kline]{}
	for _, v := range resp.List {
		if v.Time.Before(start) {
			continue
		}
		if v.Time.After(end) {
			continue
		}
		mKlines[v.Time.Format("20060102")] = append(mKlines[v.Time.Format("20060102")], v)
	}

	lss := mKlines.Sort()
	for _, ls := range lss {
		if len(ls) == 0 {
			continue
		}
		t := ls[0].Time
		data := [][]any{
			{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"}}
		for _, v := range ls {
			data = append(data, []any{
				v.Time.Format("20060102"),
				v.Time.Format("15:04"),
				v.Open.Float64(),
				v.High.Float64(),
				v.Low.Float64(),
				v.Close.Float64(),
				v.Volume,
				v.Amount.Float64(),
			})
		}
		buf, err := csv.Export(data)
		if err != nil {
			return err
		}
		err = oss.New(filepath.Join(this.Export, t.Format("20060102"), suffix, code+"-"+suffix+".csv"), buf)
		if err != nil {
			return err
		}
	}

	return nil
}
