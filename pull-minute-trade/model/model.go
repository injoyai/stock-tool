package model

import "github.com/injoyai/tdx/protocol"

type Price = protocol.Price

type Info struct {
	Date              string //日期
	TotalCapital      int64  //总股本
	NegotiableCapital int64  //流通股本
	InsideDish        int64  //内盘
	OutsideDish       int64  //外盘
}

// TotalValue 总市值,传入当日收盘价
func (this *Info) TotalValue(p int64) int64 {
	return this.TotalCapital * p
}

// NegotiableValue 流通市值,传入当日收盘价
func (this *Info) NegotiableValue(p int64) int64 {
	return this.NegotiableCapital * p
}

// TurnoverRate 换手率,传入当天成交量
// 1. 成交量/全部流通股*100%
// 2. 成交量/自由流通股*100%
func (this *Info) TurnoverRate(volume int64) float64 {
	return float64(volume) / float64(this.TotalCapital)
}
