package db

import (
	"errors"
	"fmt"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/tdx/protocol"
	"time"
)

type Sqlite struct {
	*xorms.Engine
}

func (this *Sqlite) Save(table string) {

}

// GetLast 获取最后一条数据
func (this *Sqlite) GetLast() (*Trade, error) {
	data := new(Trade)
	_, err := this.Desc("date", "time").Get(data)
	return data, err
}

// Open 打开数据库
func Open(filename string) (*Sqlite, error) {
	db, err := sqlite.NewXorm(filename)
	if err == nil {
		db.Sync2(new(Trade))
	}
	return &Sqlite{Engine: db}, err
}

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

func (this Trades) Minute1Klines() ([]*Kline, error) {
	return this.MinuteKlines(time.Minute)
}

func (this Trades) Minute5Klines() ([]*Kline, error) {
	return this.MinuteKlines(time.Minute * 5)
}

func (this Trades) MinuteKlines(interval time.Duration) ([]*Kline, error) {

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
		m[v.Time] = append(m[v.Time], v)
	}

	start1 := time.Date(0, 0, 0, 9, 30, 0, 0, time.Local)
	end1 := time.Date(0, 0, 0, 11, 30, 0, 0, time.Local).Add(1)
	start2 := time.Date(0, 0, 0, 13, 00, 0, 0, time.Local)
	end2 := time.Date(0, 0, 0, 15, 00, 0, 0, time.Local).Add(1)

	times := []string(nil)
	for t := start1; t.Before(end1); t = t.Add(interval) {
		timeStr := t.Format("15:04")
		times = append(times, timeStr)
		if _, ok := m[timeStr]; !ok {
			m[timeStr] = []*Trade{}
		}
	}

	for t := start2; t.Before(end2); t = t.Add(interval) {
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

func (this Trades) Kline(open, node int64) *Kline {

	high, low, _close := open, open, open
	volume := int64(0)
	amount := int64(0)
	for i, v := range this {
		if i == len(this)-1 {
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
		Node:      node,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     _close,
		Volume:    volume,
		Amount:    amount,
		RisePrice: _close - open,
		RiseRate:  float64(_close-open) / float64(open),
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

type Kline struct {
	Node      int64   `json:"node"`      //时间节点 2006-01-02 15:00
	Open      int64   `json:"open"`      //开盘价
	High      int64   `json:"high"`      //最高价
	Low       int64   `json:"low"`       //最低价
	Close     int64   `json:"close"`     //最新价,对应历史收盘价
	Volume    int64   `json:"volume"`    //成交量
	Amount    int64   `json:"amount"`    //成交额
	RisePrice int64   `json:"risePrice"` //涨跌幅
	RiseRate  float64 `json:"riseRate"`  //涨跌幅度
}

func (this *Kline) String() string {
	return fmt.Sprintf("%s 开盘价：%d 最高价：%d 最低价：%d 收盘价：%d 涨跌：%d 涨跌幅：%0.2f 成交量：%s 成交额：%s",
		time.Unix(this.Node, 0).Format("2006-01-02 15:04:05"),
		this.Open, this.High, this.Low, this.Close,
		this.RisePrice, this.RiseRate,
		protocol.Int64UnitString(this.Volume), protocol.Int64UnitString(this.Amount),
	)
}
