package main

import (
	"context"
	"errors"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"time"
	"xorm.io/xorm"
)

func NewConvert(codes []string, afterCode string, database, database1, database2, export string, last time.Time) *Convert {
	return &Convert{
		Database:  dir(database),
		Database1: database1,
		Database2: database2,
		Export:    export,
		Codes:     codes,
		AfterCode: afterCode,
		Last:      last,
	}
}

type Convert struct {
	Database  dir    //trade数据位置
	Database1 string //补充1
	Database2 string //补充2
	Export    string //导出位置 ./data/database/kline
	Codes     []string
	AfterCode string
	Last      time.Time
}

func (this *Convert) Run(ctx context.Context, m *tdx.Manage) error {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	for _, code := range codes {
		if code < this.AfterCode {
			continue
		}
		err := this.Database.rangeYear(code, func(year int, filename string, exist, hasNext bool) (bool, error) {
			//从23年开始存数据库,之前的直接导出
			if year < 2022 {
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

				//1分钟
				ks := data.Kline1(date, lastPrice)
				kss = append(kss, ks...)
			}
			err = this.Save(code, kss, year)
			return true, err
		})
		logs.PrintErr(err)
	}
	return nil
}

func (this *Convert) Save(code string, kss Klines, year int) error {
	m1, m5, m15, m30, m60 := kss.Merge(1), kss.Merge(5), kss.Merge(15), kss.Merge(30), kss.Merge(60)

	var err error
	m1, m5, m15, m30, m60, err = this.append(code, m1, m5, m15, m30, m60)
	if err != nil {
		return err
	}

	if err := this.save(code, m1, new(KlineMinute1), year); err != nil {
		return err
	}
	if err := this.save(code, m5, new(KlineMinute5), year); err != nil {
		return err
	}
	if err := this.save(code, m15, new(KlineMinute15), year); err != nil {
		return err
	}
	if err := this.save(code, m30, new(KlineMinute30), year); err != nil {
		return err
	}
	if err := this.save(code, m60, new(KlineMinute60), year); err != nil {
		return err
	}
	return nil
}

// save 保存数据,kss是一整年的数据
func (this *Convert) save(code string, kss Klines, table any, year int) error {
	db, err := this.open(code)
	if err != nil {
		return err
	}
	defer db.Close()
	db.Sync2(table)
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err = session.Where("Year=?", year).Delete(table); err != nil {
			return err
		}
		for _, v := range kss {
			_, err = session.Table(table).Insert(&KlineBase{
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

func (this *Convert) append(code string, m1, m5, m15, m30, m60 Klines) (Klines, Klines, Klines, Klines, Klines, error) {
	//k1补充k2
	merge := func(k1, k2 Klines) Klines {
		if len(k2) == 0 {
			return k1
		}
		if len(k1) == 0 {
			return k2
		}
		first := k2[0]
		for i, v := range k1 {
			if v.Time.Unix() == first.Time.Unix() {
				k1 = append(k1[:i-1], k2...)
				break
			}
		}
		return k1
	}
	m1_, m5_, m15_, m30_, m60_, err := this.read(code)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	m1 = merge(m1, m1_)
	m5 = merge(m5, m5_)
	m15 = merge(m15, m15_)
	m30 = merge(m30, m30_)
	m60 = merge(m60, m60_)
	return m1, m5, m15, m30, m60, nil
}

func (this *Convert) read(code string) (m1, m5, m15, m30, m60 Klines, err error) {
	k1m1, k1m5, k1m15, k1m30, k1m60, err := this.read1(code)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	k2m1, k2m5, k2m15, k2m30, k2m60, err := this.read2(code)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	//k2补充k1
	merge := func(k1, k2 Klines) Klines {
		last := &Kline{}
		if len(k1) > 0 {
			last = k1[len(k1)-1]
		}
		for i, v := range k2 {
			if v.Time.Unix() == last.Time.Unix() {
				k1 = append(k1, k2[i+1:]...)
				break
			}
		}
		return k1
	}
	k1m1 = merge(k1m1, k2m1)
	k1m5 = merge(k1m5, k2m5)
	k1m15 = merge(k1m15, k2m15)
	k1m30 = merge(k1m30, k2m30)
	k1m60 = merge(k1m60, k2m60)
	return k1m1, k1m5, k1m15, k1m30, k1m60, nil
}

func (this *Convert) read1(code string) (minute1, minute5, minute15, minute30, minute60 Klines, err error) {
	filename := filepath.Join(this.Database1, code+".db")
	if !oss.Exists(filename) {
		return Klines{}, Klines{}, Klines{}, Klines{}, Klines{}, nil
		return nil, nil, nil, nil, nil, errors.New("数据库1不存在:" + code + ".db")
	}
	db, err := sqlite.NewXorm(filepath.Join(this.Database1, code+".db"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer db.Close()

	f := func(table string) (Klines, error) {
		data := []*K1(nil)
		err = db.Table(table).Find(&data)
		if err != nil {
			return nil, err
		}
		result := Klines(nil)
		for _, v := range data {
			t := time.Unix(v.Unix, 0)
			result = append(result, &Kline{
				Time:   t,
				Open:   v.Open,
				High:   v.High,
				Low:    v.Low,
				Close:  v.Close,
				Volume: v.Volume,
				Amount: float64(v.Amount),
			})
		}
		return result, nil
	}

	minute1, err = f("KlineMinute")
	if err != nil {
		return
	}
	minute5, err = f("Kline5Minute")
	if err != nil {
		return
	}
	minute15, err = f("Kline15Minute")
	if err != nil {
		return
	}
	minute30, err = f("Kline30Minute")
	if err != nil {
		return
	}
	minute60, err = f("KlineHour")
	if err != nil {
		return
	}

	return
}

func (this *Convert) read2(code string) (minute1, minute5, minute15, minute30, minute60 Klines, err error) {
	filename := filepath.Join(this.Database2, code+".db")
	if !oss.Exists(filename) {
		return nil, nil, nil, nil, nil, errors.New("数据库2不存在:" + code + ".db")
	}
	db, err := sqlite.NewXorm(filepath.Join(this.Database2, code+".db"))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer db.Close()

	f := func(table string) (Klines, error) {
		data := []*K2(nil)
		err = db.Table(table).Find(&data)
		if err != nil {
			return nil, err
		}
		result := Klines(nil)
		for _, v := range data {
			t := time.Unix(v.Date, 0)
			result = append(result, &Kline{
				Time:   t,
				Open:   protocol.Price(v.Open).Float64(),
				High:   protocol.Price(v.High).Float64(),
				Low:    protocol.Price(v.Low).Float64(),
				Close:  protocol.Price(v.Close).Float64(),
				Volume: v.Volume,
				Amount: float64(v.Amount),
			})
		}
		return result, nil
	}

	minute1, err = f("MinuteKline")
	if err != nil {
		return
	}
	minute5, err = f("Minute5Kline")
	if err != nil {
		return
	}
	minute15, err = f("Minute15Kline")
	if err != nil {
		return
	}
	minute30, err = f("Minute30Kline")
	if err != nil {
		return
	}
	minute60, err = f("HourKline")
	if err != nil {
		return
	}

	return
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
