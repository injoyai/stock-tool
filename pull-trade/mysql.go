package main

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/database/mysql"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"time"
	"xorm.io/xorm"
)

func NewMysql(dsn string, clients, disks int) (*Mysql, error) {
	db, err := mysql.NewXorm(DSN)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	err = db.Sync2(new(TradeMysql))
	if err != nil {
		return nil, err
	}
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	if err != nil {
		return nil, err
	}
	return &Mysql{DB: db, Manage: m}, nil
}

type Mysql struct {
	DB     *xorms.Engine
	Manage *tdx.Manage
	Codes  []string
}

func (this *Mysql) Run() {
	limit := chans.NewWaitLimit(Disks)
	if len(this.Codes) == 0 {
		this.Codes = this.Manage.Codes.GetStocks()
	}
	for _, code := range this.Codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			err := g.Retry(func() error { return this.update(code) }, 3, time.Second)
			logs.PrintErr(err)
		}(code)
	}
	limit.Wait()
}

func (this *Mysql) update(code string) error {
	code = protocol.AddPrefix(code)

	last, err := this.getLast(code[2:])
	if err != nil {
		return err
	}

	lastTime := ToTime(last.Date, 0)
	now := time.Now()

	for date := lastTime; date.Before(now); date = date.Add(time.Hour * 24) {
		trades := []*TradeMysql(nil)
		err = this.Manage.Do(func(c *tdx.Client) error {
			//最早日期为2000-06-09
			if date.Before(StartDate) {
				return nil
			}
			//排除休息日
			if !this.Manage.Workday.Is(date) {
				return nil
			}
			//3. 获取数据
			trades, err = this.pullDay(c, code, date)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		logs.Debugf("%s %s %d\n", code, date.Format("2006-01-02"), len(trades))
		err = this.insert(trades)
		if err != nil {
			logs.Err(err)
			return err
		}
	}
	return nil
}

func (this *Mysql) insert(ls []*TradeMysql) error {
	return this.DB.SessionFunc(func(session *xorm.Session) error {
		for _, v := range ls {
			_, err := session.Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (this *Mysql) pullDay(c *tdx.Client, code string, start time.Time) ([]*TradeMysql, error) {

	trades := []*TradeMysql(nil)

	//获取数据的时间
	startDate, _ := FromTime(start)
	//当前时间,用于判断是否是当天
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
			trades = append(trades, &TradeMysql{
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
			trades = append(trades, &TradeMysql{
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

// 获取最后的数据,code已经处理前缀
func (this *Mysql) getLast(code string) (*TradeMysql, error) {
	//查询数据库最后的数据
	last := new(TradeMysql)
	has, err := this.DB.Where("Code=?", code).Desc("Date", "Time", "ID").Get(last)

	if err != nil {
		return nil, err
	} else if !has {
		year, month, err := getPublic(this.Manage, code)
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
		if _, err := this.DB.Where("Code=? and Date=?", code, last.Date).Delete(&TradeMysql{}); err != nil {
			return nil, err
		}
		//减去一天
		last.Date -= 1
	}
	return last, nil
}
