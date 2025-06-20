package main

import "github.com/injoyai/tdx/protocol"

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
