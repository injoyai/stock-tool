package main

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
	"xorm.io/xorm"
)

func NewConvert(codes []string, database, export string, last time.Time) *Convert {
	return &Convert{
		Database: dir(database),
		Export:   export,
		Codes:    codes,
		Last:     last,
	}
}

type Convert struct {
	Database dir
	Export   string //./data/database/kline
	Codes    []string
	Last     time.Time
}

func (this *Convert) Run(ctx context.Context, m *tdx.Manage) error {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	logs.Debug(len(codes))
	for _, code := range codes {
		err := this.Database.rangeYear(code, func(year int, filename string, exist, hasNext bool) (bool, error) {
			//从23年开始存数据库,之前的直接导出
			if year < 2023 {
				return true, nil
			}
			if !exist {
				return true, nil
			}
			logs.Debug(filename)
			db, err := sqlite.NewXorm(filename)
			if err != nil {
				return false, err
			}
			defer db.Close()
			start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
			end := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local).Add(1)
			last := conv.Select(this.Last.IsZero(), time.Now(), this.Last).Add(1)
			lastPrice := 0.
			kss := []*Kline(nil)
			for i := start; i.Before(end) && i.Before(last); i = i.Add(time.Hour * 24) {
				//排除非工作日
				if !m.Workday.Is(i) {
					continue
				}

				date, _ := FromTime(i)
				data := Trades{}
				err = db.Where("Date=?", date).Find(&data)
				if err != nil {
					return false, err
				}

				ks := data.Kline1(date, lastPrice)

				kss = append(kss, ks...)
			}
			err = this.save(code, kss, year)
			return true, err
		})
		logs.PrintErr(err)
	}
	return nil
}

// save 保存数据,kss是一整年的数据
func (this *Convert) save(code string, kss Klines, year int) error {
	db, err := this.open(code)
	if err != nil {
		return err
	}
	defer db.Close()
	db.Sync2(new(KlineMinute1))
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err = session.Where("Year=?", year).Delete(&KlineMinute1{}); err != nil {
			return err
		}
		for _, v := range kss {
			_, err = session.Insert(&KlineMinute1{
				KlineBase: KlineBase{
					Year:   v.Time.Year(),
					Month:  int(v.Time.Month()),
					Day:    v.Time.Day(),
					Hour:   v.Time.Hour(),
					Minute: v.Time.Minute(),
					Open:   v.Open,
					High:   v.High,
					Low:    v.Low,
					Close:  v.Close,
					Volume: v.Volume,
					Amount: v.Amount,
				},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (this *Convert) open(code string, year ...int) (*xorms.Engine, error) {
	if len(year) > 0 {
		return sqlite.NewXorm(filepath.Join(this.Export, code, code+"-"+conv.String(year[0])+".db"))
	}
	return sqlite.NewXorm(filepath.Join(this.Export, code+".db"))
}

/*










 */

type KlineMinute1 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute5 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute15 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute30 struct {
	KlineBase `xorm:"extends"`
}

type KlineMinute60 struct {
	KlineBase `xorm:"extends"`
}

type KlineBase struct {
	ID     int64
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int
	Amount float64
}
