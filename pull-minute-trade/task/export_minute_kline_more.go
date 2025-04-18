package task

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"time"
)

/*
ExportMinuteKlineAll
每5年一个文件

	2000-2005
	2005-2010
	2010-2015
	2015-2020

每10年一个文件

	2000-2010
	2010-2020

2020至今
*/
type ExportMinuteKlineAll struct {
	Codes       []string
	Start       time.Time
	End         time.Time
	DatabaseDir string
	OutputDir   string
	Limit       int
}

func (this *ExportMinuteKlineAll) Name() string {
	return "导出分时k线数据"
}

func (this *ExportMinuteKlineAll) Run(ctx context.Context, m *tdx.Manage) error {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	start, _ := model.FromTime(this.Start)
	end, _ := model.FromTime(this.End)

	wg := chans.NewWaitLimit(uint(this.Limit))
	for i := range codes {
		code := codes[i]

		go func(code string) {
			defer wg.Done()

			filename := filepath.Join(this.DatabaseDir, code+".db")
			if !oss.Exists(filename) {
				return
			}

			logs.Debug("开始导出:", code)
			b, err := db.Open(filename)
			if err != nil {
				logs.Err(err)
				return
			}
			defer b.Close()

			for date := start; date <= end; date++ {

				data := model.Trades{}
				err := b.Where("Date=?", date).Asc("Time").Find(&data)
				if err != nil {
					logs.Err(err)
					continue
				}

				if len(data) == 0 {
					//logs.Err("没有数据")
					continue
				}

				//生成1分钟K线
				minuteKlines, err := data.Minute1Klines()
				if err != nil {
					logs.Debug(data[0])
					logs.Err(err)
					panic(err)

					continue
				}

				logs.Debug("导出:", model.ToTime(date, 0).Format("20060102"), code, date, len(minuteKlines))
				err = klineToCsv2(code, minuteKlines, filepath.Join(this.OutputDir, code, model.ToTime(date, 0).Format("20060102")+".csv"), m.Codes.GetName)
				logs.PrintErr(err)

			}

		}(code)

	}

	wg.Wait()
	return nil
}
