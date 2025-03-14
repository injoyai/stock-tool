package model

import (
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"sort"
)

type KlineHandler func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error)

func NewKlineTable(tableName string, handler func(c *tdx.Client) KlineHandler) *KlineTable {
	return &KlineTable{
		tableName: tableName,
		Handler:   handler,
	}
}

type KlineTable struct {
	Kline     `xorm:"extends"`
	tableName string
	Handler   func(c *tdx.Client) KlineHandler `xorm:"-"`
}

func (this *KlineTable) TableName() string {
	return this.tableName
}

type IndexDayKline struct {
	Kline     `xorm:"extends"`
	Total     int //总数量
	UpCount   int //上涨数量
	DownCount int //下跌数量
}

func (this *IndexDayKline) TableName() string {
	return "DayKline"
}

type Kline struct {
	Code   string `json:"code" xorm:"-"`         //代码
	Date   int64  `json:"date"`                  //时间节点 2006-01-02 15:00
	Open   Price  `json:"open"`                  //开盘价
	High   Price  `json:"high"`                  //最高价
	Low    Price  `json:"low"`                   //最低价
	Close  Price  `json:"close"`                 //收盘价
	Volume int64  `json:"volume"`                //成交量
	Amount Price  `json:"amount"`                //成交额
	InDate int64  `json:"inDate" xorm:"created"` //创建时间
}

// RisePrice 涨跌
func (this *Kline) RisePrice() Price {
	return this.Close - this.Open
}

// RiseRate 涨跌幅
func (this *Kline) RiseRate() float64 {
	return float64(this.Close-this.Open) * 100 / float64(this.Open)
}

// Amplitude 振幅
func (this *Kline) Amplitude() float64 {
	return float64(this.High-this.Low) * 100 / float64(this.Open)
}

//func (this *Kline) String() string {
//	return fmt.Sprintf("%s 开盘：%d 最高：%d 最低：%d 收盘：%d 涨跌：%d 涨跌幅：%0.2f 成交量：%s 成交额：%s",
//		time.Unix(this.Date, 0).Format("2006-01-02 15:04:05"),
//		this.Open, this.High, this.Low, this.Close,
//		this.RisePrice(), this.RiseRate(),
//		protocol.Int64UnitString(this.Volume), protocol.FloatUnitString(this.Amount.Float64()),
//	)
//}

type Klines []*Kline

func (this Klines) Less(i, j int) bool { return this[i].Code > this[j].Code }

func (this Klines) Swap(i, j int) { this[i], this[j] = this[j], this[i] }

func (this Klines) Len() int { return len(this) }

func (this Klines) Sort() { sort.Sort(this) }

// Kline 计算多个K线,成一个K线
func (this Klines) Kline() *Kline {
	if this == nil {
		return new(Kline)
	}
	k := new(Kline)
	for i, v := range this {
		switch i {
		case 0:
			k.Open = v.Open
			k.High = v.High
			k.Low = v.Low
			k.Close = v.Close
		case len(this) - 1:
			k.Close = v.Close
			k.Date = v.Date
		}
		if v.High > k.High {
			k.High = v.High
		}
		if v.Low < k.Low {
			k.Low = v.Low
		}
		k.Volume += v.Volume
		k.Amount += v.Amount
	}

	return k
}

// Merge 合并K线
func (this Klines) Merge(n int) Klines {
	if this == nil {
		return nil
	}
	ks := []*Kline(nil)
	for i := 0; i < len(this); i += n {
		if i+n > len(this) {
			ks = append(ks, this[i:].Kline())
		} else {
			ks = append(ks, this[i:i+n].Kline())
		}
	}
	return ks
}
