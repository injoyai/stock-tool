package model

import (
	"errors"
	"time"
)

// Trade 成交数据
type Trade struct {
	Date   string `xorm:"index"` //日期
	Time   string `xorm:"index"` //时间
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

	if this[0].Time != "09:25" {
		return nil, errors.New("无效的数据源: 时间非09:25起始")
	}

	m := map[string]Trades{}
	date := this[0].Date
	for _, v := range this {
		if v.Date != date {
			return nil, errors.New("无效的数据源: 包含多个日期")
		}
		if v.Time == "09:25" {
			//通达信和东方财富,会把9.25的成交量累加到9.30里面
			v.Time = "09:30"
		}
		t, err := time.ParseInLocation("15:04", v.Time, time.Local)
		if err != nil {
			return nil, err
		}
		if v.Time != "15:00" {
			//特殊处理15:00,属于这个时间点,其他的需要加上间隔,例如09:30的成交量属于09:31
			t = t.Add(time.Minute)
		}
		timeStr := t.Format("15:04")
		m[timeStr] = append(m[timeStr], v)
	}

	start1 := time.Date(0, 0, 0, 9, 31, 0, 0, time.Local)
	end1 := time.Date(0, 0, 0, 11, 30, 0, 0, time.Local).Add(1)
	start2 := time.Date(0, 0, 0, 13, 01, 0, 0, time.Local)
	end2 := time.Date(0, 0, 0, 15, 00, 0, 0, time.Local).Add(1)

	times := []string(nil)
	for t := start1; t.Before(end1); t = t.Add(time.Minute) {
		timeStr := t.Format("15:04")
		times = append(times, timeStr)
		if _, ok := m[timeStr]; !ok {
			m[timeStr] = []*Trade{}
		}
	}

	for t := start2; t.Before(end2); t = t.Add(time.Minute) {
		timeStr := t.Format("15:04")
		times = append(times, timeStr)
		if _, ok := m[timeStr]; !ok {
			m[timeStr] = []*Trade{}
		}
	}

	klines := []*Kline(nil)
	price := this[0].Price
	for _, timeStr := range times {
		t, err := time.ParseInLocation("2006010215:04", date+timeStr, time.Local)
		if err != nil {
			return nil, err
		}
		k := m[timeStr].Kline(price, t.Unix())
		price = k.Close
		klines = append(klines, k)
	}

	return klines, nil
}

func (this Trades) Kline(last, date int64) *Kline {

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
		amount += int64(v.Volume) * v.Price
	}

	return &Kline{
		Date:   date,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  _close,
		Volume: volume,
		Amount: amount,
	}
}

// MinuteKline 用分时数据计算分钟K线
func (this Trades) MinuteKline() []*Kline {
	ls := make([]*Kline, 60*4)
	for _, v := range this {
		_ = v
	}

	return ls
}
