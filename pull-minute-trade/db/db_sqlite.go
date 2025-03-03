package db

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"pull-minute-trade/model"
)

type Sqlite struct {
	*xorms.Engine
}

// GetLastTrade 获取最后一条分时数据
func (this *Sqlite) GetLastTrade() (*model.Trade, error) {
	data := new(model.Trade)
	_, err := this.Desc("Date", "Time").Get(data)
	return data, err
}

// GetLastKline 获取最后一条K线数据
func (this *Sqlite) GetLastKline() (*model.Kline, error) {
	data := new(model.Kline)
	_, err := this.Desc("Date").Get(data)
	return data, err
}

// Open 打开数据库
func Open(filename string) (*Sqlite, error) {
	db, err := sqlite.NewXorm(filename)
	if err == nil {
		db.Sync2(new(model.Trade))
		db.Sync2(new(model.DayKline))
	}
	return &Sqlite{Engine: db}, err
}
