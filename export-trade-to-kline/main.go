package main

import (
	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx/protocol"
	"os"
	"path/filepath"
	"xorm.io/xorm"
)

var (
	DatabaseDir = cfg.GetString("database_dir", "./data/database/trade")
	ExportDir   = cfg.GetString("export_dir", "./data/database/kline")
	Coroutine   = cfg.GetInt("coroutine", 10)
)

func main() {

	es, err := os.ReadDir(DatabaseDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutine, bar.WithPrefix("[xx000000]"))

	for i := range es {
		e := es[i]

		b.Go(func() {
			if !e.IsDir() {
				return
			}

			b.SetPrefix("[" + e.Name() + "]")
			b.Flush()

			dir := filepath.Join(DatabaseDir, e.Name())
			vs, err := os.ReadDir(dir)
			if err != nil {
				b.Logf("[ERR] [%s] %v", dir, err)
				b.Flush()
				return
			}

			for _, v := range vs {
				err = convert(
					filepath.Join(dir, v.Name()),
					filepath.Join(ExportDir, e.Name(), v.Name()),
				)
				if err != nil {
					b.Logf("[ERR] [%s] %v", e.Name(), err)
					b.Flush()
					return
				}
			}

		})
	}

	b.Wait()

}

func convert(tradFilename string, klineFilename string) error {
	if !oss.Exists(tradFilename) {
		return nil
	}
	db, err := sqlite.NewXorm(tradFilename)
	if err != nil {
		return err
	}
	defer db.Close()

	data := []*Trade(nil)
	err = db.Find(&data)
	if err != nil {
		return err
	}

	ts := protocol.Trades{}
	for _, v := range data {
		ts = append(ts, &protocol.Trade{
			Time:   ToTime(v.Date, v.Time),
			Price:  v.Price,
			Volume: v.Volume,
			Status: v.Status,
		})
	}

	ks := ts.Klines()

	if len(ks) == 0 {
		return nil
	}

	return insert(klineFilename, ks)
}

func insert(klineFilename string, ks protocol.Klines) error {
	db, err := sqlite.NewXorm(klineFilename)
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Sync2(new(Kline))
	if err != nil {
		return err
	}

	inserts := []*Kline(nil)
	for _, k := range ks {
		inserts = append(inserts, &Kline{
			Year:   k.Time.Year(),
			Month:  int(k.Time.Month()),
			Day:    k.Time.Day(),
			Hour:   k.Time.Hour(),
			Minute: k.Time.Minute(),
			Open:   k.Open.Float64(),
			High:   k.High.Float64(),
			Low:    k.Low.Float64(),
			Close:  k.Close.Float64(),
			Volume: k.Volume,
			Amount: k.Amount.Float64(),
		})
	}
	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range inserts {
			_, err = session.Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
