package main

import (
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/mysql"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/str/bar"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"time"
	"xorm.io/xorm"
)

var (
	StartDate = time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local)
)

var (
	DB     *xorms.Engine
	Manage *tdx.Manage
	Codes  = []string{
		"sz000001",
		//"sh600000",
	}
)

func init() {
	var err error
	DB, err = mysql.NewXorm(cfg.GetString("database.dsn"))
	logs.PanicErr(err)
	logs.PanicErr(DB.Ping())
	DB.Sync2(new(Trade))
}

func main() {

	var err error
	Manage, err = tdx.NewManage(&tdx.ManageConfig{Number: 4})
	logs.PanicErr(err)

	limit := chans.NewWaitLimit(100)
	if len(Codes) == 0 {
		Codes = Manage.Codes.GetStocks()
	}

	b := bar.New(int64(len(Codes) * (time.Now().Year() - StartDate.Year() + 1)))
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

	for _, code := range Codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			err := update(code, func(n int) { b.Add(int64(n)).Flush() })
			logs.PrintErr(err)
		}(code)
	}
	limit.Wait()
}

func update(code string, add func(n int)) error {
	last, err := getLast(code)
	if err != nil {
		return err
	}

	lastTime := ToTime(last.Date, 0)
	add(lastTime.Year() - StartDate.Year())
	now := time.Now()

	for date := lastTime; date.Before(now); date = date.Add(time.Hour * 24) {
		logs.Debug(date)
		trades := []*Trade(nil)
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
			trades, err = pullDay(c, code, date)
			if err != nil {
				return err
			}
			logs.Debug(3)
			//trades = append(trades, item...)
			//logs.Debug(4)
			return nil
		})
		if err != nil {
			return err
		}
		logs.Debug(5)
		logs.Debug(len(trades))
		err = insert(trades)
		if err != nil {
			logs.Err(err)
			return err
		}
		add(1)
	}
	return nil
}

func insert(ls []*Trade) error {
	return DB.SessionFunc(func(session *xorm.Session) error {
		for _, v := range ls {
			_, err := session.Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func pullDay(c *tdx.Client, code string, start time.Time) ([]*Trade, error) {

	logs.Spend(code + start.Format("-20060102") + "耗时")()

	trades := []*Trade(nil)

	startDate, _ := FromTime(start)

	nowDate, _ := FromTime(time.Now())

	switch startDate {
	case 0:
	//

	case nowDate:
		//获取当天数据
		resp, err := c.GetTradeAll(code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			date, minute := FromTime(v.Time)
			trades = append(trades, &Trade{
				Exchange: code[:2],
				Code:     code[2:],
				Show:     v.Time.Format(time.DateTime),
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
			date, minute := FromTime(v.Time)
			trades = append(trades, &Trade{
				Exchange: code[:2],
				Code:     code[2:],
				Show:     v.Time.Format("2006-01-02 15:04"),
				Date:     date,
				Time:     minute,
				Price:    v.Price,
				Volume:   v.Volume,
				Order:    0,
				Status:   v.Status,
			})
		}

	}

	return trades, nil
}

func getLast(code string) (*Trade, error) {
	//查询数据库最后的数据
	last := new(Trade)
	has, err := DB.Where("Code=?", code).Desc("Date").Desc("Date", "Time").Get(last)
	if err != nil {
		return nil, err
	} else if !has {
		year, month, err := getPublic(Manage, code)
		if err != nil {
			return nil, err
		}
		date := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		if date.Before(StartDate) {
			date = StartDate
		}
		//说明数据不存在,取该股上市月初为起始时间
		last.Date, _ = FromTime(date)
	} else if last.Time != 900 {
		//如果最后时间不是15:00,说明数据不全,删除这天的数据
		if _, err := DB.Where("Code=?", code).And("Date=?", last.Date).Delete(&Trade{}); err != nil {
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
