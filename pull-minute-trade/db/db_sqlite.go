package db

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
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
	Date   string //日期
	Time   string //时间
	Price  int64  //成交价格,单位分
	Volume int    //交易量
	Order  int    //订单数
	Status int    //买或者卖
}

type Trades []*Trade

// MinuteKline 用分时数据计算分钟K线
func (this Trades) MinuteKline() []*Kline {
	ls := make([]*Kline, 60*4)
	for _, v := range this {
		_ = v
	}

	return ls
}

type Kline struct {
	Code      string  `json:"code" xorm:"index"` //代码
	Unix      int64   `json:"unix"`              //时间戳
	Open      float64 `json:"open"`              //开盘价
	High      float64 `json:"high"`              //最高价
	Low       float64 `json:"low"`               //最低价
	Close     float64 `json:"close"`             //最新价,对应历史收盘价
	Volume    int64   `json:"volume"`            //成交量
	Amount    float64 `json:"amount"`            //成交额
	RisePrice float64 `json:"risePrice"`         //涨跌幅
	RiseRate  float64 `json:"riseRate"`          //涨跌幅度
}
