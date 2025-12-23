package main

import (
	"time"

	"github.com/injoyai/tdx/protocol"
)

type Trade struct {
	Date   uint16         `xorm:"index"` //日期
	Time   uint16         //时间 `xorm:"index"` //时间
	Price  protocol.Price //成交价格,单位厘
	Volume int            //交易量
	Order  int            //订单数
	Status int            //买或者卖
}

func (this *Trade) To() *protocol.Trade {
	return &protocol.Trade{
		Time:   ToTime(this.Date, this.Time),
		Price:  this.Price,
		Volume: this.Volume,
		Status: this.Status,
		Number: 0,
	}
}

// ToTime 转时间,最大支持170年,即1990+170=2160
func ToTime(date, minute uint16) time.Time {
	yearMonth := date >> 5
	year := int(yearMonth/12) + 1990
	month := time.Month(yearMonth%12 + 1)
	day := int(date & 31)
	return time.Date(year, month, day, int(minute/60), int(minute%60), 0, 0, time.Local)
}

// FromTime x
func FromTime(t time.Time) (date uint16, minute uint16) {
	return (uint16(t.Year()-1990)*12+uint16(t.Month()-1))<<5 + uint16(t.Day()), uint16(t.Hour()*60 + t.Minute())
}
