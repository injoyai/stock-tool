package main

import "time"

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
	Volume int
	Amount float64
}
