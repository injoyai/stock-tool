package main

import (
	"context"
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

func NewPullKlineDay(codes []string, export string) *PullKlineDay {
	return &PullKlineDay{
		Export: export,
		Codes:  codes,
	}
}

type PullKlineDay struct {
	Export string
	Codes  []string
}

func (this *PullKlineDay) Run(ctx context.Context, m *tdx.Manage) error {

	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	total := int64(0)
	for i := xx; i.Before(times.IntegerDay(time.Now())); i = i.AddDate(0, 0, 1) {
		if m.Workday.Is(i) {
			total++
		}
	}

	b := bar.New()
	b.SetTotal(total)
	limit := chans.NewWaitLimit(100)
	for i := xx; i.Before(times.IntegerDay(time.Now())); i = i.AddDate(0, 0, 1) {
		if m.Workday.Is(i) {
			limit.Add()
			go func() {
				defer limit.Done()
				defer func() {
					b.Add(1)
					b.Flush()
				}()
				err := g.Retry(func() error {
					return this.pullAll(codes, i, m)
				}, 3, time.Second*2)
				logs.PrintErr(err)
			}()
		}
	}
	limit.Wait()

	return nil
}

func (this *PullKlineDay) pullAll(codes []string, i time.Time, m *tdx.Manage) error {
	err := this.pull(
		m, codes, i, "1min",
		func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error) {
			return c.GetKlineMinuteUntil
		},
	)
	if err != nil {
		return err
	}

	err = this.pull(
		m, codes, i, "5min",
		func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error) {
			return c.GetKline5MinuteUntil
		},
	)
	if err != nil {
		return err
	}

	err = this.pull(
		m, codes, i, "15min",
		func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error) {
			return c.GetKline15MinuteUntil
		},
	)
	if err != nil {
		return err
	}

	err = this.pull(
		m, codes, i, "30min",
		func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error) {
			return c.GetKline30MinuteUntil
		},
	)
	if err != nil {
		return err
	}

	err = this.pull(
		m, codes, i, "60min",
		func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error) {
			return c.GetKlineHourUntil
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (this *PullKlineDay) pull(m *tdx.Manage, codes []string, date time.Time, suffix string,
	fn func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error)) error {

	startDate := times.IntegerDay(date)
	endDate := times.IntegerDay(date).AddDate(0, 0, 1).Add(-1)
	data := [][]any{
		{"Code", "Date", "Time", "Open", "High", "Low", "Close", "Volume", "Amount"},
	}
	for _, code := range codes {
		err := m.Do(func(c *tdx.Client) error {
			resp, err := fn(c)(code, func(k *protocol.Kline) bool {
				return k.Time.Before(startDate)
			})
			if err != nil {
				return err
			}
			if len(resp.List) > 0 {
				for _, v := range resp.List {
					if v.Time.Before(startDate) {
						continue
					}
					if v.Time.After(endDate) {
						continue
					}
					data = append(data, []any{
						code,
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
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	return oss.New(filepath.Join(this.Export, date.Format("20060102"), date.Format("20060102")+"-"+suffix+".csv"), buf)
}

func (this *PullKlineDay) pullDay(m *tdx.Manage, codes []string, start, end time.Time) error {
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
				return this.pullCodes(code, start, end, m)
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

func (this *PullKlineDay) pullCodes(code string, start, end time.Time, m *tdx.Manage) error {
	err := this.pullCode(code, start, end, "1min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKlineMinuteAll(code)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, start, end, "5min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline5MinuteAll(code)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, start, end, "15min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline15MinuteAll(code)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, start, end, "30min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKline30MinuteAll(code)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	err = this.pullCode(code, start, end, "60min", func(code string) (resp *protocol.KlineResp, err error) {
		err = m.Do(func(c *tdx.Client) error {
			resp, err = c.GetKlineHourAll(code)
			return err
		})
		return
	})
	if err != nil {
		return err
	}

	return nil
}

func (this *PullKlineDay) pullCode(code string, start, end time.Time, suffix string, fn func(code string) (*protocol.KlineResp, error)) error {
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

type cache struct {
	Code   string
	Offset uint16
	Count  uint16
	Cache  []*protocol.Kline
	m      *tdx.Manage
}

func (this *cache) Get1(offset uint16) (ls []*protocol.Kline, err error) {
	err = this.m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineMinute(this.Code, offset, this.Count)
		if err != nil {
			return err
		}
		ls = resp.List
		return nil
	})
	return
}

//func (this *PullKlineDay) pull(m *tdx.Manage, code string, suffix string, fn func(c *tdx.Client) func(code string) (*protocol.KlineResp, error)) error {
//
//	mKlines := types.SortMap[string, []*protocol.Kline]{}
//
//	err := m.Do(func(c *tdx.Client) error {
//		resp, err := fn(c)(code)
//		if err != nil {
//			return err
//		}
//		for _, v := range resp.List {
//			mKlines[v.Time.Format("20060102")] = append(mKlines[v.Time.Format("20060102")], v)
//		}
//		return nil
//	})
//	if err != nil {
//		return err
//	}
//
//	lss := mKlines.Sort()
//	for _, ls := range lss {
//		if len(ls) == 0 {
//			continue
//		}
//		t := ls[0].Time
//		if t.Before(xx) {
//			continue
//		}
//		data := [][]any{
//			{"Code", "Date", "Time", "Open", "High", "Low", "Close", "Volume", "Amount"},
//		}
//		for _, v := range ls {
//			data = append(data, []any{
//				code,
//				v.Time.Format("20060102"),
//				v.Time.Format("15:04"),
//				v.Open.Float64(),
//				v.High.Float64(),
//				v.Low.Float64(),
//				v.Close.Float64(),
//				v.Volume,
//				v.Amount.Float64(),
//			})
//		}
//		buf, err := csv.Export(data)
//		if err != nil {
//			return err
//		}
//		err = oss.New(filepath.Join(
//			this.Export,
//			t.Format("20060102"),
//			suffix,
//			code+".csv",
//		), buf)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//

var (
	xx = time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
)
