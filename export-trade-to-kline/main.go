package main

import (
	"os"
	"path/filepath"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

var (
	TradeDir  = cfg.GetString("database_dir", "./data/database/trade")
	KlineDir  = cfg.GetString("export_dir", "./data/database/kline")
	Coroutine = cfg.GetInt("coroutine", 30)
	After     = cfg.GetString("after", "")
)

func main() {

	es, err := os.ReadDir(TradeDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(len(es), Coroutine, bar.WithPrefix("[xx000000]"))

	for i := range es {
		e := es[i]

		b.Go(func() {

			if e.Name() < After {
				return
			}

			if !e.IsDir() {
				return
			}

			b.SetPrefix("[" + e.Name() + "]")
			b.Flush()

			dir := filepath.Join(TradeDir, e.Name())
			vs, err := os.ReadDir(dir)
			if err != nil {
				b.Logf("[ERR] [%s] %v", dir, err)
				b.Flush()
				return
			}

			for _, v := range vs {
				err = convert(
					filepath.Join(dir, v.Name()),
					filepath.Join(KlineDir, e.Name(), v.Name()),
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
	err = db.Sync2(new(protocol.Kline))
	if err != nil {
		return err
	}

	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range ks {
			_, err = session.Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
