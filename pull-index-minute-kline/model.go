package main

import (
	"github.com/injoyai/tdx/protocol"
	"time"
)

type KlineMinute1 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute5 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute15 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute30 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute60 struct {
	KlineBase `xorm:"extends"`
}

type Timer interface {
	Time() time.Time
}

type KlineBase struct {
	ID     int64
	Date   int64
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
	Amount float64
}

type Klines []*KlineBase

func (this Klines) Merge(n int) Klines {
	ls := protocol.Klines{}
	for _, v := range this {
		ls = append(ls, &protocol.Kline{
			Open:   protocol.Price(v.Open * 1000),
			High:   protocol.Price(v.High * 1000),
			Low:    protocol.Price(v.Low * 1000),
			Close:  protocol.Price(v.Close * 1000),
			Volume: v.Volume,
			Amount: protocol.Price(v.Amount * 1000),
			Time:   time.Unix(v.Date, 0),
		})
	}
	ls = ls.Merge(n)
	ks := Klines{}
	for _, v := range ls {
		ks = append(ks, &KlineBase{
			Date:   v.Time.Unix(),
			Year:   v.Time.Year(),
			Month:  int(v.Time.Month()),
			Day:    v.Time.Day(),
			Hour:   v.Time.Hour(),
			Minute: v.Time.Minute(),
			Open:   v.Open.Float64(),
			High:   v.High.Float64(),
			Low:    v.Low.Float64(),
			Close:  v.Close.Float64(),
			Volume: v.Volume,
			Amount: v.Amount.Float64(),
		})
	}
	return ks
}
