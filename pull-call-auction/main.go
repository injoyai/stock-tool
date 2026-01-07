package main

import (
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/lib/xorms"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"xorm.io/xorm"
)

var (
	Spec        = cfg.GetString("spec", "20 0 15 * * *")
	Goroutines  = cfg.GetInt("goroutines", 20)
	Startup     = cfg.GetBool("startup")
	Codes       = cfg.GetStrings("codes")
	DatabaseDir = cfg.GetString("database", "./data/database/auction")
)

func main() {

	m, err := tdx.NewManage()
	logs.PanicErr(err)

	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(Spec, func() {
		logs.PrintErr(update(m, Codes, Goroutines))
	})

	if Startup {
		logs.PrintErr(update(m, Codes, Goroutines))
	}

	cr.Run()

}

func update(m *tdx.Manage, codes []string, goroutines int) error {

	year := conv.String(time.Now().Year())

	if len(codes) == 0 {
		codes = m.Codes.GetStockCodes()
	}

	b := bar.NewCoroutine(len(codes), goroutines)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.Go(func() {
			err := g.Retry(func() error {
				return m.Do(func(c *tdx.Client) error {
					return pull(c, DatabaseDir, year, code)
				})
			}, tdx.DefaultRetry)
			if err != nil {
				b.Logf("[错误] [%s] %s\n", code, err)
				b.Flush()
			}
		})
	}

	b.Wait()

	return nil
}

func pull(c *tdx.Client, dir, year, code string) error {

	todayNode := tdx.IntegerDay(time.Now())

	//只能盘后更新
	if time.Now().Before(todayNode.Add(time.Minute * 60 * 15)) {
		return nil
	}

	filename := filepath.Join(dir, code, code+"-"+year+".db")
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	db.Sync2(new(protocol.CallAuction))

	last := new(protocol.CallAuction)
	_, err = db.Desc("Time").Get(last)
	if err != nil {
		return err
	}

	if last.Time.After(todayNode) {
		return nil
	}

	resp, err := c.GetCallAuction(code)
	if err != nil {
		return err
	}

	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			if _, err = session.Insert(v); err != nil {
				return err
			}
		}
		return nil
	})

}
