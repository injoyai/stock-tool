package main

import "github.com/injoyai/tdx/protocol"

type Trade struct {
	ID       int64
	Exchange string         //交易所
	Code     string         `xorm:"index"` //代码
	Show     string         //日期可视化
	Date     uint16         `xorm:"index"` //日期
	Time     uint16         `xorm:"index"` //时间
	Price    protocol.Price //成交价格,单位厘
	Volume   int            //交易量
	Order    int            //订单数
	Status   int            //0买,1卖,2
}
