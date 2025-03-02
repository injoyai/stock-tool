package main

import (
	"github.com/injoyai/logs"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
)

/*

根据分时数据计算出分线

*/

func main() {

	logs.SetFormatter(logs.TimeFormatter)

	date := "20250227"

	b, err := db.Open("./data/database/tdx/trade/sz000001.db")
	logs.PanicErr(err)

	data := model.Trades{}
	err = b.Where("Date=?", date).Asc("Time").Find(&data)
	logs.PanicErr(err)

	ks, err := data.MinuteKlines()
	logs.PanicErr(err)

	for _, v := range ks {
		logs.Debug(v)
	}

}
