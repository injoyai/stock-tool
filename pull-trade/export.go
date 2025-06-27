package main

import (
	"bytes"
	"context"
	"github.com/injoyai/base/types"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/tdx"
	"time"
)

func NewExport(codes []string, years []int, database, export string) *Export {
	return &Export{
		Database:  dir(database),
		Codes:     codes,
		Years:     years,
		Export:    export,
		ReadLimit: 10,
	}
}

type Export struct {
	Database  dir
	Codes     types.List[string]
	Years     []int
	Export    string
	ReadLimit int
	HHD[any, any]
}

func (this *Export) Run(ctx context.Context, m *tdx.Manage) error {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	//for _, year := range this.Years {
	//	for _, cs := range codes.Split(this.ReadLimit) {
	//		this.HHD.Run(ctx)
	//	}
	//}
	return nil
}

func (this *Export) read(year int, codes []string) ([]any, error) {
	//for _, v := range codes {
	//
	//}
	return nil, nil
}

func (this *Export) deal() {
	//db, err := sqlite.NewXorm(filename)
	//if err != nil {
	//	return err
	//}
	//defer db.Close()
	//start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	//end := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local).Add(1)
	//last := 0.
	//kss := []*Kline(nil)
	//for i := start; i.Before(end); i = i.Add(time.Hour * 24) {
	//	if !m.Workday.Is(i) {
	//		continue
	//	}
	//	date, _ := FromTime(i)
	//	data := Trades{}
	//	err = db.Where("Date=?", date).Find(&data)
	//	if err != nil {
	//		return false, err
	//	}
	//	ks := data.Kline1(date, last)
	//	kss = append(kss, ks...)
	//}
	//xx := [][]any(nil)
	//for _, v := range kss {
	//	xx = append(xx, []any{
	//		v.Time.Format(time.DateTime),
	//		v.Open,
	//		v.High,
	//		v.Low,
	//		v.Close,
	//		v.Volume,
	//		v.Amount,
	//	})
	//}
	//buf, err := csv.Export(xx)
	//if err != nil {
	//	return false, err
	//}
	//oss.New(filepath.Join(this.Export, conv.String(year), "k线-1分钟", code+".csv"), buf)
	//return true, nil
}

func (this *Export) save(t *exportTask) (err error) {
	err = oss.New(t.Filename, t.K1)
	if err != nil {
		return err
	}
	err = oss.New(t.Filename, t.K5)
	if err != nil {
		return err
	}
	err = oss.New(t.Filename, t.K15)
	if err != nil {
		return err
	}
	err = oss.New(t.Filename, t.K30)
	if err != nil {
		return err
	}
	err = oss.New(t.Filename, t.K60)
	if err != nil {
		return err
	}
	return
}

type exportTask struct {
	Filename string
	K1       *bytes.Buffer
	K5       *bytes.Buffer
	K15      *bytes.Buffer
	K30      *bytes.Buffer
	K60      *bytes.Buffer
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
		k := ls.Kline(last.Time, ls[0].Open)
		ks = append(ks, k)
	}
	return ks
}
