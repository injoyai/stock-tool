package main

import (
	"context"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
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

	b := bar.New(func(b bar.Base) {
		b.SetTotal(int64(len(codes)))
		b.SetFormat(func(b bar.Bar) string {
			return fmt.Sprintf("\r[更新] %s  %s  %s",
				b.Plan(),
				b.RateSize(),
				b.Speed(),
			)
		})
	})

	limit := chans.NewWaitLimit(this.Limit)
	for _, code := range codes {
		limit.Add()
		go func(code string) {
			defer limit.Done()
			defer func() {
				b.Add(1)
				b.Flush()
			}()
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

	now := time.Now()
	node := times.IntegerDay(now)
	//顺序写入硬盘
	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			if !v.Time.After(lastTime) {
				continue
			}
			//判断今天是否在15点之后,否则取消今天的数据入库
			if now.Hour() < 15 && v.Time.After(node) {
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
