package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"xorm.io/xorm"
)

func NewUpdateKline(codes []string, database string, limit int) *UpdateKline {
	return &UpdateKline{
		Database: database,
		Codes:    codes,
		Limit:    limit,
	}
}

type UpdateKline struct {
	Database string
	Codes    []string
	Limit    int
}

func (this *UpdateKline) Run(ctx context.Context, m *tdx.Manage) error {

	codes := this.Codes
	if len(this.Codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	limit := chans.NewWaitLimit(this.Limit)
	for _, code := range codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			err := this.update(code, m)
			logs.PrintErr(err)
		}(code)
	}
	limit.Wait()

	return nil
}

func (this *UpdateKline) update(code string, c *tdx.Manage) error {

	//打开数据库
	filename := filepath.Join(this.Database, code+".db")
	logs.Debug(filename)
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	logs.PrintErr(db.Sync2(new(KlineMinute1)))
	logs.PrintErr(db.Sync2(new(KlineMinute5)))
	logs.PrintErr(db.Sync2(new(KlineMinute15)))
	logs.PrintErr(db.Sync2(new(KlineMinute30)))
	logs.PrintErr(db.Sync2(new(KlineMinute60)))

	err = this.pull(db, c, protocol.TypeKlineMinute, code, new(KlineMinute1))
	logs.PrintErr(err)
	err = this.pull(db, c, protocol.TypeKline5Minute, code, new(KlineMinute5))
	logs.PrintErr(err)
	err = this.pull(db, c, protocol.TypeKline15Minute, code, new(KlineMinute15))
	logs.PrintErr(err)
	err = this.pull(db, c, protocol.TypeKline30Minute, code, new(KlineMinute30))
	logs.PrintErr(err)
	err = this.pull(db, c, protocol.TypeKlineHour, code, new(KlineMinute60))
	logs.PrintErr(err)

	return nil
}

func (this *UpdateKline) pull(db *xorms.Engine, c *tdx.Manage, _type uint8, code string, last Timer) error {
	//读取数据库最后一条数据
	_, err := db.Desc("ID").Get(last)
	if err != nil {
		logs.Err(err)
		return err
	}
	lastTime := last.Time()
	logs.Debug(lastTime.String())

	//拉取数据
	var resp *protocol.KlineResp
	err = c.Do(func(c *tdx.Client) error {
		resp, err = c.GetKlineUntil(_type, code, func(k *protocol.Kline) bool {
			return k.Time.Before(lastTime)
		})
		return err
	})
	if err != nil {
		return err
	}

	//顺序写入硬盘
	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			if !v.Time.After(lastTime) {
				continue
			}
			_, err = session.Table(last).Insert(&KlineBase{
				Year:   v.Time.Year(),
				Month:  int(v.Time.Month()),
				Day:    v.Time.Day(),
				Hour:   v.Time.Hour(),
				Minute: v.Time.Minute(),
				Open:   v.Open.Float64(),
				High:   v.High.Float64(),
				Low:    v.Low.Float64(),
				Close:  v.Close.Float64(),
				Volume: int(v.Volume),
				Amount: v.Amount.Float64(),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
