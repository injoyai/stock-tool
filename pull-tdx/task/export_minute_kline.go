package task

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"
	"time"
)

func NewExportMinuteKline(codes []string, databaseDir, minute1Dir, minute5Dir string, limit int) *ExportMinuteKline {
	return &ExportMinuteKline{
		Codes:       codes,
		databaseDir: databaseDir,
		minute1Dir:  minute1Dir,
		minute5Dir:  minute5Dir,
		Limit:       limit,
	}
}

type ExportMinuteKline struct {
	Codes       []string
	databaseDir string
	minute1Dir  string
	minute5Dir  string
	Limit       int
}

func (this *ExportMinuteKline) Name() string {
	return "导出分时k线数据"
}

func (this *ExportMinuteKline) Run(ctx context.Context, m *tdx.Manage) error {
	now := time.Now()
	date, _ := model.FromTime(now)

	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	//logs.Debug(codes)

	wg := chans.NewWaitLimit(uint(this.Limit))

	for i := range codes {
		code := codes[i]

		wg.Add()
		go func(code string) {
			defer wg.Done()

			filename := filepath.Join(this.databaseDir, code+".db")
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

			data := model.Trades{}
			err = b.Where("Date=?", date).Asc("Time").Find(&data)
			if err != nil {
				logs.Err(err)
				return
			}

			if len(data) == 0 {
				logs.Err("没有数据")
				return
			}

			//导出1分钟K线
			minuteKlines, err := data.Minute1Klines()
			if err != nil {
				logs.Err(err)
				return
			}
			{
				ls := [][]any{
					{"日期", "代码", "名称", "开盘", "最高", "最低", "收盘", "总手", "金额", "涨幅", "涨幅比"},
				}
				for _, v := range minuteKlines {
					ls = append(ls, []any{
						time.Unix(v.Date, 0).Format(time.DateTime), code, m.Codes.GetName(code),
						v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(),
						v.Volume, v.Amount.Float64(), v.RisePrice().Float64(), v.RiseRate()},
					)
				}
				buf, err := excel.ToCsv(ls)
				if err != nil {
					logs.Err(err)
					return
				}
				err = oss.New(filepath.Join(this.minute1Dir, code, now.Format("20060102")+".csv"), buf)
				if err != nil {
					logs.Err(err)
					return
				}
			}

			{ //导出5分钟K线
				minute5Klines := minuteKlines.Merge(5)
				ls := [][]any{
					{"日期", "代码", "名称", "开盘", "最高", "最低", "收盘", "总手", "金额", "涨幅", "涨幅比"},
				}
				for _, v := range minute5Klines {
					ls = append(ls, []any{time.Unix(v.Date, 0).Format(time.DateTime), code, m.Codes.GetName(code),
						v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(),
						v.Volume, v.Amount.Float64(), v.RisePrice().Float64(), v.RiseRate()})
				}
				buf, err := excel.ToCsv(ls)
				if err != nil {
					logs.Err(err)
					return
				}
				err = oss.New(filepath.Join(this.minute5Dir, code, now.Format("20060102")+".csv"), buf)
				if err != nil {
					logs.Err(err)
					return
				}
			}

		}(code)

	}

	wg.Wait()

	return nil
}
