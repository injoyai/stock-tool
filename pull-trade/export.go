package main

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

func NewExport(codes []string, database, export string) *Export {
	return &Export{
		Database: dir(database),
		Codes:    codes,
		Export:   export,
	}
}

type Export struct {
	Database dir
	Codes    []string
	Export   string
}

func (this *Export) Run(ctx context.Context, m *tdx.Manage) {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	for _, code := range this.Codes {
		this.Database.rangeYear(code, func(year int, filename string, exist, hasNext bool) (bool, error) {
			db, err := sqlite.NewXorm(filename)
			if err != nil {
				return false, err
			}
			defer db.Close()
			start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
			end := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local).Add(1)
			last := 0.
			kss := []*Kline(nil)
			for i := start; i.Before(end); i = i.Add(time.Hour * 24) {
				if !m.Workday.Is(i) {
					continue
				}
				date, _ := FromTime(i)
				data := Trades{}
				err = db.Where("Date=?", date).Find(&data)
				if err != nil {
					return false, err
				}
				ks := data.Kline1(date, last)
				kss = append(kss, ks...)
			}
			xx := [][]any(nil)
			for _, v := range kss {
				xx = append(xx, []any{
					v.Time.Format(time.DateTime),
					v.Open,
					v.High,
					v.Low,
					v.Close,
					v.Volume,
					v.Amount,
				})
			}
			buf, err := csv.Export(xx)
			if err != nil {
				return false, err
			}
			oss.New(filepath.Join(this.Export, conv.String(year), "k线-1分钟", code+".csv"), buf)
			return true, nil
		})
	}

}

/*


















 */

type Kline struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int
	Amount float64
}

type Klines []*Kline

func (this Klines) Kline(t time.Time, last float64) *Kline {
	k := &Kline{
		Time:   t,
		Open:   last,
		High:   last,
		Low:    last,
		Close:  last,
		Volume: 0,
		Amount: 0,
	}
	for i, v := range this {
		switch i {
		case 0:
			k.Open = v.Open
			k.High = v.High
			k.Low = v.Low
			k.Close = v.Close
		default:
			k.High = conv.Select(k.High < v.High, v.High, k.High)
			k.Low = conv.Select(k.Low > v.Low, v.Low, k.Low)
		}
		k.Close = v.Close
		k.Volume += v.Volume
		k.Amount += v.Amount
	}
	return k
}

// Merge 合并成其他类型的K线
func (this Klines) Merge(n int) Klines {
	if n <= 1 {
		return this
	}
	ks := Klines(nil)
	ls := Klines(nil)
	for i := 0; ; i++ {
		if len(this) <= i*n {
			break
		}
		if len(this) < (i+1)*n {
			ls = this[i*n:]
		} else {
			ls = this[i*n : (i+1)*n]
		}
		if len(ls) == 0 {
			break
		}
		last := ls[len(ls)-1]
		ls.Kline(last.Time, ls[0].Open)
	}
	return ks
}
