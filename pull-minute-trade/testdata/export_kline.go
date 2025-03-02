package main

import (
	"github.com/injoyai/logs"
	"pull-minute-trade/db"
)

/*

根据分时数据计算出分线

*/

func main() {

	logs.SetFormatter(logs.TimeFormatter)

	date := "20250227"

	b, err := db.Open("./data/database/trade/sz000001.db")
	logs.PanicErr(err)

	data := db.Trades{}
	err = b.Where("Date=?", date).Asc("Time").Find(&data)
	logs.PanicErr(err)

	ks, err := data.Minute5Klines()
	logs.PanicErr(err)

	for _, v := range ks {
		logs.Debug(v)
	}

}
