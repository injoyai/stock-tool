package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/tdx/protocol"
)

type TradeMysql struct {
	ID       int64
	Exchange string         //交易所
	Code     string         `xorm:"index"` //代码
	Date     uint16         `xorm:"index"` //日期
	Time     uint16         `xorm:"index"` //时间
	Show     string         //日期可视化,后续会删除
	Price    protocol.Price //成交价格,单位厘
	Volume   int            //交易量
	Order    int            //订单数
	Status   int            //0买,1卖,2
}

func (this TradeMysql) TableName() string {
	return "trade"
}

// TradeSqlite 成交数据
type TradeSqlite struct {
	Date   uint16         `xorm:"index"` //日期
	Time   uint16         //时间 `xorm:"index"` //时间
	Price  protocol.Price //成交价格,单位厘
	Volume int            //交易量
	Order  int            //订单数
	Status int            //买或者卖
}

func (this TradeSqlite) TableName() string {
	return "trade"
}

/*














 */

type Trades []*TradeSqlite

func (this Trades) Kline1(date uint16, last float64) []*Kline {
	_930 := uint16(570)
	_1130 := uint16(690)
	_1300 := uint16(780)
	_1500 := uint16(900)
	keys := []uint16(nil)
	//早上
	m := map[uint16]Trades{}
	for i := uint16(1); i <= 120; i++ {
		keys = append(keys, _930+i)
		m[_930+i] = []*TradeSqlite{}
	}
	//下午
	for i := uint16(1); i <= 120; i++ {
		keys = append(keys, _1300+i)
		m[_1300+i] = []*TradeSqlite{}
	}
	//分组
	for _, v := range this {
		t := conv.Select(v.Time <= _930, _930, v.Time)
		t++
		t = conv.Select(t > _1130 && t <= _1300, _1130, t)
		t = conv.Select(t > _1500, _1500, t)
		m[t] = append(m[t], v)
	}
	//合并
	ls := []*Kline(nil)
	for _, v := range keys {
		k := m[v].Merge(date, v, last)
		last = k.Close
		ls = append(ls, k)
	}
	return ls
}

// Merge 合并分时成交成k线
func (this Trades) Merge(date, time uint16, last float64) *Kline {
	k := &Kline{
		Time:  ToTime(date, time),
		Open:  last,
		High:  last,
		Low:   last,
		Close: last,
	}
	for i, v := range this {
		switch i {
		case 0:
			k.Open = v.Price.Float64()
			k.High = v.Price.Float64()
			k.Low = v.Price.Float64()
			k.Close = v.Price.Float64()
		default:
			k.High = conv.Select(k.High < v.Price.Float64(), v.Price.Float64(), k.High)
			k.Low = conv.Select(k.Low > v.Price.Float64(), v.Price.Float64(), k.Low)
		}
		k.Close = v.Price.Float64()
		k.Volume += v.Volume
		k.Amount += v.Price.Float64() * float64(v.Volume) * 100
	}
	return k
}
