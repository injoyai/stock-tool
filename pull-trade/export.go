package main

import (
	"context"
	"github.com/injoyai/base/types"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

func NewExport(codes []string, years []int, database, export string) *Export {
	return &Export{
		Database:  dir(database),
		Codes:     codes,
		Years:     years,
		Export:    export,
		ReadSplit: 10,
	}
}

type Export struct {
	Database  dir
	Codes     types.List[string]
	Years     []int
	Export    string
	ReadSplit int
}

func (this *Export) Run(ctx context.Context, m *tdx.Manage) error {
	codes := this.Codes
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}

	for _, year := range this.Years {
		logs.Debugf("执行年份: %d\n", year)
		for _, cs := range codes.Split(this.ReadSplit) {
			tasks, err := this.Read(year, cs)
			if err != nil {
				logs.Err(err)
				continue
			}
			for _, task := range tasks {
				err := task.Save()
				logs.PrintErr(err)
			}
		}
	}
	return nil
}

func (this *Export) Read(year int, codes []string) ([]*exportTask, error) {
	tasks := []*exportTask(nil)
	for _, code := range codes {
		err := func() error {
			filename := this.Database.filename(code, year)
			db, err := sqlite.NewXorm(filename)
			if err != nil {

				return err
			}
			defer db.Close()
			data := Trades(nil)
			if err = db.Find(&data); err != nil {
				return err
			}
			if len(data) == 0 {
				return nil
			}
			//按天分割
			m := make(Map[uint16, Trades])
			for _, v := range data {
				m[v.Date] = append(m[v.Date], v)
			}
			mKline := Map[uint16, Klines]{}
			for date, v := range m {
				mKline[date] = v.Kline1(date, 0)
			}
			ls := mKline.Sort()
			var k1 Klines
			for _, v := range ls {
				k1 = append(k1, v...)
			}
			t := &exportTask{
				Code: code,
				Dir:  filepath.Join(this.Export, conv.String(year)),
				K1:   k1,
				K5:   k1.Merge(5),
				K15:  k1.Merge(15),
				K30:  k1.Merge(30),
				K60:  k1.Merge(60),
			}
			tasks = append(tasks, t)

			return nil
		}()
		logs.PrintErr(err)
	}

	return tasks, nil
}

type exportTask struct {
	Code string
	Dir  string
	K1   Klines
	K5   Klines
	K15  Klines
	K30  Klines
	K60  Klines
}

func (this *exportTask) Filename(typeName string) string {
	return filepath.Join(this.Dir, typeName, this.Code+".csv")
}

func (this *exportTask) Save() error {
	if err := this.save(this.K1, "1分钟"); err != nil {
		return err
	}
	if err := this.save(this.K5, "5分钟"); err != nil {
		return err
	}
	if err := this.save(this.K15, "15分钟"); err != nil {
		return err
	}
	if err := this.save(this.K30, "30分钟"); err != nil {
		return err
	}
	if err := this.save(this.K60, "60分钟"); err != nil {
		return err
	}
	return nil
}

func (this *exportTask) save(ks Klines, typeName string) error {
	data := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
	}
	for _, v := range ks {
		data = append(data, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
			v.Open,
			v.High,
			v.Low,
			v.Close,
			v.Volume,
			float64(int64(v.Amount)),
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	return oss.New(this.Filename(typeName), buf)
}

/*


















 */

type Kline struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int
	Amount float64
}

type Klines []*Kline

func (this Klines) Kline(t time.Time, last float64) *Kline {
	k := &Kline{
		Time:   t,
		Open:   last,
		High:   last,
		Low:    last,
		Close:  last,
		Volume: 0,
		Amount: 0,
	}
	for i, v := range this {
		switch i {
		case 0:
			k.Open = v.Open
			k.High = v.High
			k.Low = v.Low
			k.Close = v.Close
		default:
			k.High = conv.Select(k.High < v.High, v.High, k.High)
			k.Low = conv.Select(k.Low > v.Low, v.Low, k.Low)
		}
		k.Close = v.Close
		k.Volume += v.Volume
		k.Amount += v.Amount
	}
	return k
}

// Merge 合并成其他类型的K线
func (this Klines) Merge(n int) Klines {
	if n <= 1 {
		return this
	}
	ks := Klines(nil)
	ls := Klines(nil)
	for i := 0; ; i++ {
		if len(this) <= i*n {
			break
		}
		if len(this) < (i+1)*n {
			ls = this[i*n:]
		} else {
			ls = this[i*n : (i+1)*n]
		}
		if len(ls) == 0 {
			break
		}
		last := ls[len(ls)-1]
		k := ls.Kline(last.Time, ls[0].Open)
		ks = append(ks, k)
	}
	return ks
}
