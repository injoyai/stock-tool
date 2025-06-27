package main

import (
	"context"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
)

type Merge struct {
	Codes     []string
	Database1 string
	Database2 string
}

func (this *Merge) Run(ctx context.Context, m *tdx.Manage) error {

	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	for _, code := range codes {
		_ = code
		ls1, err := this.read1(code)
		if err != nil {
			logs.Err(err)
			continue
		}
		ls2, err := this.read2(code)
		if err != nil {
			logs.Err(err)
			continue
		}
		last := &KlineBase{}
		if len(ls1) > 0 {
			last = ls1[len(ls1)-1]
		}
		for i, v := range ls2 {
			if v.Year == last.Year && v.Month == last.Month && v.Day == last.Day && v.Hour == last.Hour && v.Minute == last.Minute {
				ls1 = append(ls1, ls2[i+1:]...)
				break
			}
		}

	}

	return nil
}

func (this *Merge) append(ls []*KlineBase) {

}

func (this *Merge) read1(code string) ([]*KlineBase, error) {
	db, err := sqlite.NewXorm(filepath.Join(this.Database1, code+".db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	data := []*K1(nil)
	err = db.Find(&data)
	if err != nil {
		return nil, err
	}
	result := []*KlineBase(nil)
	for _, v := range data {
		t := time.Unix(v.Unix, 0)
		result = append(result, &KlineBase{
			Year:   t.Year(),
			Month:  int(t.Month()),
			Day:    t.Day(),
			Hour:   t.Hour(),
			Minute: t.Minute(),
			Open:   v.Open,
			High:   v.High,
			Low:    v.Low,
			Close:  v.Close,
			Volume: v.Volume,
			Amount: float64(v.Amount),
		})
	}
	return result, nil
}

func (this *Merge) read2(code string) ([]*KlineBase, error) {
	db, err := sqlite.NewXorm(filepath.Join(this.Database2, code+".db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	data := []*K2(nil)
	err = db.Find(&data)
	if err != nil {
		return nil, err
	}
	result := []*KlineBase(nil)
	for _, v := range data {
		t := time.Unix(v.Date, 0)
		result = append(result, &KlineBase{
			Year:   t.Year(),
			Month:  int(t.Month()),
			Day:    t.Day(),
			Hour:   t.Hour(),
			Minute: t.Minute(),
			Open:   protocol.Price(v.Open).Float64(),
			High:   protocol.Price(v.High).Float64(),
			Low:    protocol.Price(v.Low).Float64(),
			Close:  protocol.Price(v.Close).Float64(),
			Volume: v.Volume,
			Amount: float64(v.Amount),
		})
	}
	return result, nil
}

/*














 */

type K1 struct {
	Unix   int64
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int
	Amount int
}

type K2 struct {
	Date   int64
	Open   int
	High   int
	Low    int
	Close  int
	Volume int
	Amount int64
}
