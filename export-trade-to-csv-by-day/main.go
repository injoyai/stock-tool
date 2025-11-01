package main

import (
	"fmt"
	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	DatabaseDir = cfg.GetString("database_dir", "./data/database/trade")
	ExportDir   = cfg.GetString("export_dir", "./data/export_by_day")
	Coroutine   = cfg.GetInt("coroutine", 10)
	StartYear   = cfg.GetInt("start_year", 1990)
	EndYear     = cfg.GetInt("end_year", time.Now().Year())
)

func init() {
	os.MkdirAll(ExportDir, 0777)
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("数据库目录:", DatabaseDir)
	logs.Info("导出目录:", ExportDir)
	logs.Info("并发协程:", Coroutine)
	logs.Infof("年份范围: %d-%d\n", StartYear, EndYear)
	fmt.Println("============================================")
}

func main() {

	defer g.InputEnterFunc()()

	c, err := tdx.DialDefault()
	logs.PanicErr(err)

	w, err := tdx.NewWorkdaySqlite(c)
	logs.PanicErr(err)

	cs, err := os.ReadDir(DatabaseDir)
	logs.PanicErr(err)

	b := bar.NewCoroutine(
		len(cs),
		Coroutine,
		bar.WithPrefix("[xx000000]"),
		bar.WithFinal(func(b *bar.Bar) {
			logs.Info("任务完成...")
		}),
	)
	defer b.Close()

	now := time.Now()

	for i := range cs {
		v := cs[i]
		b.Go(func() {
			b.SetPrefix("[" + v.Name() + "]")
			b.Flush()

			if !v.IsDir() {
				return
			}
			for year := StartYear; year <= EndYear && year <= time.Now().Year(); year++ {

				w.Range(
					time.Date(year, 1, 1, 0, 0, 0, 0, time.Local),
					time.Date(year, 12, 31, 0, 0, 1, 0, time.Local),
					func(t time.Time) bool {
						if t.After(now) {
							return false
						}
						dir := filepath.Join(DatabaseDir, v.Name())
						dbname := v.Name() + "-" + conv.String(year) + ".db"
						err = export(dir, dbname, t)
						if err != nil {
							b.Logf("[ERR] [%s] %s\n", v.Name(), err)
						}
						return true
					},
				)

			}

		})

	}

	b.Wait()

}

func export(dir string, dbname string, t time.Time) error {
	name := strings.Split(dbname, "-")[0]
	filename := filepath.Join(dir, dbname)
	if !oss.Exists(filename) {
		return nil
	}

	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	//defer db.Close()

	date, _ := FromTime(t)
	data := []*Trade(nil)
	err = db.Where("Date=?", date).Find(&data)
	db.Close()
	if err != nil {
		return err
	}

	output := filepath.Join(ExportDir, t.Format("2006/20060102"), name+".csv")
	return save(data, output)
}

func save(ts []*Trade, output string) error {
	data := [][]any{
		{"时间", "价格", "成交量", "方向(0买,1卖,2中性)"},
	}
	for _, v := range ts {
		t := ToTime(v.Date, v.Time)
		data = append(data, []any{
			t.Format(time.DateTime),
			v.Price.Float64(),
			v.Volume,
			v.Status,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	return oss.New(output, buf)
}
