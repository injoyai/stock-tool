package main

type DayKline struct {
	ID       int64
	Code     string `xorm:"index"`
	Date     int64  `xorm:"index"`
	Year     int
	Month    int
	Day      int
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   int64
	Amount   float64
	InDate   int64 `xorm:"created"`
	EditDate int64 `xorm:"updated"`
}
