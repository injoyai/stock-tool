package main

import (
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"time"
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.SetShowColor(false)
}

func main() {
	defer done()()

	startDate := time.Date(2025, 2, 17, 0, 0, 0, 0, time.Local)
	endDate := time.Date(2025, 3, 31, 23, 0, 0, 0, time.Local)
	size := 200

	{
		var err error
		for {
			start := g.Input("请输入开始时间(例2025-02-17): ")
			startDate, err = time.Parse("2006-01-02 15:04:05", start+" 00:00:00")
			if err == nil {
				break
			}
			logs.Err(err)
		}
		for {
			end := g.InputVar("请输入结束时间(默认今天): ").String(time.Now().Format(time.DateOnly))
			endDate, err = time.Parse("2006-01-02 15:04:05", end+" 23:00:00")
			if err == nil {
				break
			}
			logs.Err(err)
		}
		size = g.InputVar("请数据每个文件代码数量(默认6000):").Int(6000)
		logs.Info("准备开始下载数据...")
	}

	bySector(
		[]string{},
		startDate,
		endDate,
		size,
	)

	//byStock(
	//	[]string{},
	//	time.Date(2016, 1, 1, 0, 0, 0, 0, time.Local),
	//	time.Date(2025, 3, 23, 23, 0, 0, 0, time.Local),
	//)

}
