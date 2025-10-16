package main

import (
	"github.com/injoyai/base/types"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/mysql"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

const (
	Clients   = 2
	Coroutine = 20
)

func main() {

	dsn := cfg.GetString("database.dsn")

	db, err := mysql.NewXorm(dsn)
	logs.PanicErr(err)

	err = db.Sync2(new(DayKline))
	logs.PanicErr(err)

	m, err := tdx.NewManageMysql(&tdx.ManageConfig{
		Number:          Clients,
		CodesFilename:   dsn,
		WorkdayFileName: dsn,
	})
	logs.PanicErr(err)

	Run(m, db.Engine)

}

func Run(m *tdx.Manage, db *xorm.Engine) {
	codes := m.Codes.GetStocks()
	b := bar.NewCoroutine(len(codes), Coroutine)
	defer b.Close()
	for i, _ := range codes {
		code := codes[i]
		b.Go(func() {
			var (
				ks  []*protocol.Kline
				err error
			)
			err = m.Do(func(c *tdx.Client) error {
				ks, err = pull(c, db, code)
				return err
			})
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err)
				b.Flush()
				return
			}
			if err := update(db, ks, code); err != nil {
				b.Logf("[ERR] [%s] %s", code, err)
				b.Flush()
			}
		})
	}
	b.Wait()
}

func pull(c *tdx.Client, db *xorm.Engine, code string) ([]*protocol.Kline, error) {
	//2. 获取最后一条数据
	last := new(DayKline)
	if _, err := db.Where("Code=?", code).Desc("Date").Get(last); err != nil {
		return nil, err
	}

	if last.Date == 0 {
		last.Date = protocol.ExchangeEstablish.Unix()
	}

	//3. 从服务器获取数据
	resp, err := c.GetKlineDayUntil(code, func(k *protocol.Kline) bool {
		return k.Time.Unix() <= last.Date
	})
	if err != nil {
		return nil, err
	}

	return resp.List, nil
}

func update(db *xorm.Engine, ks []*protocol.Kline, code string) error {
	inserts := []*DayKline(nil)
	for _, v := range ks {
		inserts = append(inserts, &DayKline{
			Code:   code,
			Date:   v.Time.Unix(),
			Year:   v.Time.Year(),
			Month:  int(v.Time.Month()),
			Day:    v.Time.Day(),
			Open:   v.Open.Float64(),
			High:   v.High.Float64(),
			Low:    v.Low.Float64(),
			Close:  v.Close.Float64(),
			Volume: v.Volume,
			Amount: v.Amount.Float64(),
		})
	}

	if len(inserts) == 0 {
		return nil
	}

	//4. 插入数据库
	if _, err := db.Table(new(DayKline)).Where("Code=? AND Date >= ?", code, inserts[0].Date).Delete(); err != nil {
		return err
	}

	ls := types.List[any](conv.Array(inserts))
	for _, v := range ls.Split(3000) {
		if _, err := db.Insert(v); err != nil {
			return err
		}
	}

	return nil
}
