package main

import (
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/str/bar"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"time"
)

var (
	StartDate = time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local)
)

var (
	DB     *xorms.Engine
	Manage *tdx.Manage
)

func init() {
	var err error
	DB, err = xorms.NewMysql(cfg.GetString("database.dsn"))
	logs.PanicErr(err)
	Manage, err = tdx.NewManage(nil)
	logs.PanicErr(err)
}

func main() {

	limit := chans.NewWaitLimit(100)
	codes := Manage.Codes.GetStocks()

	b := bar.New(int64(len(codes) * (time.Now().Year() - StartDate.Year() + 1)))
	b.AddOption(func(f *bar.Format) {
		f.Entity.SetFormatter(func(e *bar.Format) string {
			return fmt.Sprintf("\r%s [%s] %s  %s  %s  %-10s",
				time.Now().Format(time.TimeOnly),
				"进度",
				e.Bar,
				e.RateSize,
				e.Speed,
				e.Used,
			)
		})
	})
	b.Add(0).Flush()

	for _, code := range codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			err := update(code, func(n int) { b.Add(int64(n)).Flush() })
			logs.PrintErr(err)
		}(code)
	}

}

func update(code string, add func(n int)) error {
	last, err := getLast(code)
	if err != nil {
		return err
	}

	lastTime := ToTime(last.Date, 0)
	add(lastTime.Year() - StartDate.Year())
	now := time.Now()
	insert := []*StockTrade(nil)
	for date := lastTime; date.Before(now); date = date.Add(time.Hour * 24) {
		err = Manage.Do(func(c *tdx.Client) error {
			//最早日期为2000-06-09
			if date.Before(StartDate) {
				return nil
			}

			//排除休息日
			if !Manage.Workday.Is(date) {
				return nil
			}

			//3. 获取数据
			item, err := pullDay(c, code, date)
			if err != nil {
				return err
			}
			insert = append(insert, item...)
			return nil
		})
		if err != nil {
			return err
		}
		if len(insert) > 1000 {
			_, err = DB.Insert(insert)
			if err != nil {
				return err
			}
		}
		add(1)
	}
	if len(insert) > 0 {
		_, err = DB.Insert(insert)
		if err != nil {
			return err
		}
	}
	return nil
}

func pullDay(c *tdx.Client, code string, start time.Time) ([]*StockTrade, error) {

	insert := []*StockTrade(nil)

	date, _ := FromTime(start)

	nowDate, _ := FromTime(time.Now())

	switch date {
	case 0:
	//

	case nowDate:
		//获取当天数据
		resp, err := c.GetTradeAll(code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := FromTime(v.Time)
			insert = append(insert, &StockTrade{
				Exchange: code[:2],
				Code:     code[2:],
				Show:     start.Format(time.DateTime),
				Date:     date,
				Time:     minute,
				Price:    v.Price,
				Volume:   v.Volume,
				Order:    v.Number,
				Status:   v.Status,
			})
		}

	default:
		//获取历史数据
		resp, err := c.GetHistoryTradeAll(start.Format("20060102"), code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := FromTime(v.Time)
			insert = append(insert, &StockTrade{
				Exchange: code[:2],
				Code:     code[2:],
				Show:     start.Format(time.DateTime),
				Date:     date,
				Time:     minute,
				Price:    v.Price,
				Volume:   v.Volume,
				Order:    0,
				Status:   v.Status,
			})
		}

	}

	return insert, nil
}

func getLast(code string) (*StockTrade, error) {
	//查询数据库最后的数据
	last := new(StockTrade)
	has, err := DB.Where("Code=?", code).Desc("Date").Desc("Data", "Time").Get(last)
	if err != nil {
		return nil, err
	} else if !has {
		year, month, err := getPublic(Manage, code)
		if err != nil {
			return nil, err
		}
		//说明数据不存在,取该股上市月初为起始时间
		last.Date, _ = FromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.Local))
	} else if last.Time != 900 {
		//如果最后时间不是15:00,说明数据不全,删除这天的数据
		if _, err := DB.Where("Code=?", code).And("Date=?", last.Date).Delete(&StockTrade{}); err != nil {
			return nil, err
		}
		//减去一天
		last.Date -= 1
	}
	return last, nil
}

func getPublic(m *tdx.Manage, code string) (year int, month time.Month, err error) {
	year = 1990
	month = 12
	err = m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineMonthAll(code)
		if err != nil {
			return err
		}
		if len(resp.List) > 0 {
			year = resp.List[0].Time.Year()
			month = resp.List[0].Time.Month()
			return nil
		}
		return nil
	})
	return
}
