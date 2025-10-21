package main

import (
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

//func (Trade) TableName() string {
//	return "trade"
//}
