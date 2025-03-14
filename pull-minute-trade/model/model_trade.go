package model

import (
	"errors"
	"time"
)

// Trade 成交数据
type Trade struct {
	Date   uint16 `xorm:"index"` //日期
	Time   uint16 //时间 `xorm:"index"` //时间
	Price  int64  //成交价格,单位分
	Volume int    //交易量
	Order  int    //订单数
	Status int    //买或者卖
}

type Trades []*Trade

func (this Trades) Minute1Klines() (Klines, error) {
	return this.MinuteKlines()
}

func (this Trades) Minute5Klines() (Klines, error) {
	ks, err := this.MinuteKlines()
	return ks.Merge(5), err
}

func (this Trades) Minute15Klines() (Klines, error) {
	ks, err := this.MinuteKlines()
	return ks.Merge(15), err
}

func (this Trades) Minute30Klines() (Klines, error) {
	ks, err := this.MinuteKlines()
	return ks.Merge(30), err
}

func (this Trades) HourKlines() (Klines, error) {
	ks, err := this.MinuteKlines()
	return ks.Merge(60), err
}

func (this Trades) DayKlines() (Klines, error) {
	ks, err := this.MinuteKlines()
	return ks.Merge(len(ks)), err
}

func (this Trades) MinuteKlines() (Klines, error) {

	if len(this) == 0 {
		return nil, errors.New("无效的数据源: 为空")
	}

	if this[0].Time != 565 { // "09:25" {
		return nil, errors.New("无效的数据源: 时间非09:25起始")
	}

	m := map[uint16]Trades{}
	date := this[0].Date
	for _, v := range this {
		if v.Date != date {
			return nil, errors.New("无效的数据源: 包含多个日期")
		}
		if v.Time == 565 { // "09:25"
			//通达信和东方财富,会把9.25的成交量累加到9.30里面
			v.Time = 570 //"09:30"
		}
		if v.Time != 900 { //"15:00"
			//特殊处理15:00,属于这个时间点,其他的需要加上间隔,例如09:30的成交量属于09:31
			v.Time += 1
		}
		m[v.Time] = append(m[v.Time], v)
	}

	start1 := time.Date(0, 0, 0, 9, 31, 0, 0, time.Local)
	end1 := time.Date(0, 0, 0, 11, 30, 0, 0, time.Local).Add(1)
	start2 := time.Date(0, 0, 0, 13, 01, 0, 0, time.Local)
	end2 := time.Date(0, 0, 0, 15, 00, 0, 0, time.Local).Add(1)

	minutes := []uint16(nil)
	for t := start1; t.Before(end1); t = t.Add(time.Minute) {
		_, minute := FromTime(t)
		minutes = append(minutes, minute)
		if _, ok := m[minute]; !ok {
			m[minute] = []*Trade{}
		}
	}

	for t := start2; t.Before(end2); t = t.Add(time.Minute) {
		_, minute := FromTime(t)
		minutes = append(minutes, minute)
		if _, ok := m[minute]; !ok {
			m[minute] = []*Trade{}
		}
	}

	klines := []*Kline(nil)
	price := this[0].Price
	for _, minute := range minutes {
		t := ToTime(date, minute)
		k := m[minute].Kline(price, t.Unix())
		price = k.Close.Int64()
		klines = append(klines, k)
	}

	return klines, nil
}

func (this Trades) Kline(last int64, date int64) *Kline {

	open, high, low, _close := last, last, last, last
	volume := int64(0)
	amount := int64(0)
	for i, v := range this {
		switch i {
		case 0:
			open = v.Price
			high = v.Price
			low = v.Price
			_close = v.Price
		case len(this) - 1:
			_close = v.Price
		}
		if v.Price > high {
			high = v.Price
		}
		if v.Price < low {
			low = v.Price
		}
		volume += int64(v.Volume)
		amount += int64(v.Volume) * 100 * v.Price
	}

	return &Kline{
		Date:   date,
		Open:   Price(open),
		High:   Price(high),
		Low:    Price(low),
		Close:  Price(_close),
		Volume: volume,
		Amount: Price(amount),
	}
}
