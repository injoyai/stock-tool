package main

import (
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/times"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

func Update(m *tdx.Manage, codes []string, year string, goroutines int, dir string) error {

	b := bar.NewCoroutine(len(codes), goroutines, bar.WithPrefix("[更新]"))
	defer b.Close()

	for _, code := range codes {
		b.Go(func() {
			err := update(m, dir, code, year)
			logs.PrintErr(err)
		})
	}
	b.Wait()

	return nil
}

func update(c *tdx.Manage, dir, code, year string) error {

	//打开数据库
	filename := filepath.Join(dir, code, code+"-"+year+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	logs.PrintErr(db.Sync2(new(protocol.Kline)))

	//读取数据库最后一条数据
	last := new(protocol.Kline)
	_, err = db.Desc("Time").Get(last)
	if err != nil {
		logs.Err(err)
		return err
	}

	//logs.Debug()
	//logs.Debug(filename)
	//logs.Debug("最后一条数据:", last)

	//拉取数据
	var resp *protocol.KlineResp
	err = c.Do(func(c *tdx.Client) error {
		resp, err = c.GetKlineMinute241Until(code, func(k *protocol.Kline) bool {
			return k.Time.Before(last.Time)
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
			if !v.Time.After(last.Time) {
				continue
			}
			//判断今天是否在15点之后,否则取消今天的数据入库
			if now.Hour() < 15 && v.Time.After(node) {
				continue
			}
			_, err = session.Table(last).Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})

}
