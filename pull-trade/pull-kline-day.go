package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/types"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
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

	wg := chans.NewWaitLimit(100)
	for i := range codes {
		code := codes[i]
		logs.Debug(code)
		wg.Add()
		go func(code string) {
			defer wg.Done()

			err := this.pull(
				m, code, "1min",
				func(c *tdx.Client) func(code string) (*protocol.KlineResp, error) {
					return c.GetKlineMinuteAll
				},
			)
			if err != nil {
				logs.Err(err)
				return
			}

			err = this.pull(
				m, code, "5min",
				func(c *tdx.Client) func(code string) (*protocol.KlineResp, error) {
					return c.GetKline5MinuteAll
				},
			)
			if err != nil {
				logs.Err(err)
				return
			}

			err = this.pull(
				m, code, "15min",
				func(c *tdx.Client) func(code string) (*protocol.KlineResp, error) {
					return c.GetKline15MinuteAll
				},
			)
			if err != nil {
				logs.Err(err)
				return
			}

			err = this.pull(
				m, code, "30min",
				func(c *tdx.Client) func(code string) (*protocol.KlineResp, error) {
					return c.GetKline30MinuteAll
				},
			)
			if err != nil {
				logs.Err(err)
				return
			}

			err = this.pull(
				m, code, "60min",
				func(c *tdx.Client) func(code string) (*protocol.KlineResp, error) {
					return c.GetKlineHourAll
				},
			)
			if err != nil {
				logs.Err(err)
				return
			}

		}(code)

	}

	wg.Wait()
	logs.Debug("pull done")

	return oss.RangeFileInfo(this.Export, func(info *oss.FileInfo) (bool, error) {
		if info.IsDir() {
			return true, zip.Encode(
				info.FullName(),
				info.FullName()+".zip",
			)
		}
		return true, nil
	})

	//return zip.Encode(
	//	this.Export,
	//	this.Export+".zip",
	//)

}

//func (this *PullKlineDay) pull(codes []string, date time.Time, m *tdx.Manage,
//	fn func(c *tdx.Client) func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error),
//	suffix string) error {
//	startDate := times.IntegerDay(date)
//	endDate := times.IntegerDay(date).AddDate(0, 0, 1).Add(-1)
//	data := [][]any{
//		{"Code", "Date", "Time", "Open", "High", "Low", "Close", "Volume", "Amount"},
//	}
//	for _, code := range codes {
//		err := m.Do(func(c *tdx.Client) error {
//			resp, err := fn(c)(code, func(k *protocol.Kline) bool {
//				return k.Time.Before(startDate)
//			})
//			if err != nil {
//				return err
//			}
//			if len(resp.List) > 0 {
//				for _, v := range resp.List {
//					if v.Time.Before(startDate) {
//						continue
//					}
//					if v.Time.After(endDate) {
//						continue
//					}
//					data = append(data, []any{
//						code,
//						v.Time.Format("20060102"),
//						v.Time.Format("15:04"),
//						v.Open.Float64(),
//						v.High.Float64(),
//						v.Low.Float64(),
//						v.Close.Float64(),
//						v.Volume,
//						v.Amount.Float64(),
//					})
//				}
//			}
//			return nil
//		})
//		if err != nil {
//			return err
//		}
//	}
//
//	buf, err := csv.Export(data)
//	if err != nil {
//		return err
//	}
//	return oss.New(filepath.Join(this.Export, date.Format("20060102"), date.Format("20060102")+"-"+suffix+".csv"), buf)
//}

func (this *PullKlineDay) pull(m *tdx.Manage, code string, suffix string, fn func(c *tdx.Client) func(code string) (*protocol.KlineResp, error)) error {

	mKlines := types.SortMap[string, []*protocol.Kline]{}

	err := m.Do(func(c *tdx.Client) error {
		resp, err := fn(c)(code)
		if err != nil {
			return err
		}
		for _, v := range resp.List {
			mKlines[v.Time.Format("20060102")] = append(mKlines[v.Time.Format("20060102")], v)
		}
		return nil
	})
	if err != nil {
		return err
	}

	lss := mKlines.Sort()
	for _, ls := range lss {
		if len(ls) == 0 {
			continue
		}
		t := ls[0].Time
		if t.Before(xx) {
			continue
		}
		data := [][]any{
			{"Code", "Date", "Time", "Open", "High", "Low", "Close", "Volume", "Amount"},
		}
		for _, v := range ls {
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
		buf, err := csv.Export(data)
		if err != nil {
			return err
		}
		err = oss.New(filepath.Join(
			this.Export,
			t.Format("20060102"),
			suffix,
			code+".csv",
		), buf)
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	xx = time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
)
