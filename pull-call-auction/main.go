package main

import (
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
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
	ExportDir   = cfg.GetString("export", "./data/export/")
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

	defer func() {
		logs.Info("任务完成...")
	}()

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
					return pull(c, DatabaseDir, year, code, ExportDir)
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

func pull(c *tdx.Client, dir, year, code string, exportDir string) error {

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

	data := []*protocol.CallAuction(nil)
	err = db.Where("Time>?", todayNode).Find(&data)
	if err != nil {
		return err
	}

	if len(data) > 0 && data[0].Time.After(todayNode) {
		return export(data, todayNode, exportDir, code)
	}

	resp, err := c.GetCallAuction(code)
	if err != nil {
		return err
	}

	err = db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range resp.List {
			if _, err = session.Insert(v); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return export(resp.List, todayNode, exportDir, code)
}

func export(ls []*protocol.CallAuction, todayNode time.Time, exportDir string, code string) error {
	exportFilename := filepath.Join(exportDir, conv.String(todayNode.Year()), todayNode.Format(time.DateOnly), code+".csv")
	data := [][]any{
		{"时间", "价格", "匹配量(股)", "未匹配量(股)", "未匹配量类型(1买单,-1卖单)"},
	}
	for _, v := range ls {
		data = append(data, []any{
			v.Time.Format(time.DateTime),
			v.Price.Float64(),
			v.Match * 100,
			v.Unmatched * 100,
			v.Flag,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}

	return oss.New(exportFilename, buf)
}
