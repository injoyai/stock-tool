package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
)

const (
	DefaultRetry = 3
)

var (
	Clients          = cfg.GetInt("clients", 4)
	Coroutines       = cfg.GetInt("coroutines", 10)
	ExportCoroutines = cfg.GetInt("export_coroutines", 20)
	Tasks            = cfg.GetInt("tasks", 2)
	DatabaseDir      = cfg.GetString("database", "./data/database")
	ExportDir        = cfg.GetString("export", "./data/export")
	UploadDir        = cfg.GetString("upload", "./data/upload")
	Spec             = cfg.GetString("spec", "0 10 15 * * *")
	Codes            = cfg.GetStrings("codes")
	Startup          = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.10")
	logs.Info("说明:", "修复非工作日执行的bug")
	logs.Info("任务规则:", Spec)
	logs.Info("立马执行:", Startup)
	logs.Info("连接数量:", Clients)
	logs.Info("协程数量1:", Coroutines)
	logs.Info("协程数量2:", ExportCoroutines)
	fmt.Println("=====================================================")
	os.MkdirAll(DatabaseDir, 0755)
	os.MkdirAll(ExportDir, 0755)
	os.MkdirAll(UploadDir, 0755)
}

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() {
		if !m.Workday.TodayIs() {
			logs.Error("今天不是工作日")
			return
		}
		run(m, Codes)
	})
	if Startup {
		run(m, Codes)
	}
	t.Run()
}

func run(m *tdx.Manage, codes []string) {
	if len(codes) == 0 {
		codes = m.Codes.GetStocks()
	}
	logs.PrintErr(update(m, codes))
	//logs.PrintErr(exportThisYear(m, codes))
	logs.PrintErr(exportThisDay(codes))
}

func update(m *tdx.Manage, codes []string) error {
	logs.Info("[更新] 最新数据...")
	return NewSqlite(
		codes,
		filepath.Join(DatabaseDir, "trade"),
		Coroutines,
		Tasks,
	).Run(context.Background(), m)
}

func exportThisYear(m *tdx.Manage, codes []string) error {

	logs.Info("[导出] 本年数据...")

	year := conv.String(time.Now().Year())
	os.MkdirAll(filepath.Join(ExportDir, year), 0755)
	os.MkdirAll(filepath.Join(UploadDir, year), 0755)

	b := bar.NewCoroutine(len(codes), ExportCoroutines,
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		filename := filepath.Join(DatabaseDir, "trade", code, code+"-"+year+".db")
		b.Go(func() {

			b.SetPrefix("[" + code + "]")
			b.Flush()

			db, err := sqlite.NewXorm(filename)
			if err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}
			defer db.Close()

			//读取当前全部数据
			var trades []*Trade
			err = db.Find(&trades)
			if err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}

			//导出
			output := filepath.Join(ExportDir, year, code+".csv")
			if err = save(trades, output); err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}

		})
	}

	b.Wait()

	//压缩
	logs.Info("[导出] 本年数据压缩...")
	zipFilename := filepath.Join(ExportDir, year+".zip")
	err := zip.Encode(
		filepath.Join(ExportDir, year),
		zipFilename,
	)
	if err != nil {
		return err
	}

	//重命名
	logs.Info("[导出] 本年数据重命名...")
	<-time.After(time.Second * 5)
	err = os.Rename(zipFilename, filepath.Join(UploadDir, year, year+".zip"))
	if err != nil {
		return err
	}

	logs.Info("[导出] 本年数据完成...")

	return nil
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

func exportThisDay(codes []string) error {

	logs.Info("[导出] 今日数据...")

	now := time.Now()
	day := now.Format("20060102")
	year := conv.String(now.Year())
	os.MkdirAll(filepath.Join(ExportDir, year), 0755)
	os.MkdirAll(filepath.Join(ExportDir, "day"), 0755)
	os.MkdirAll(filepath.Join(UploadDir, year, "每日数据"), 0755)

	date, _ := FromTime(now)

	b := bar.NewCoroutine(len(codes), ExportCoroutines,
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		filename := filepath.Join(DatabaseDir, "trade", code, code+"-"+year+".db")
		b.Go(func() {

			b.SetPrefix("[" + code + "]")
			b.Flush()

			db, err := sqlite.NewXorm(filename)
			if err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}
			defer db.Close()

			//读取当前全部数据
			var trades []*Trade
			err = db.Where("Date=?", date).Find(&trades)
			if err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}

			//导出
			output := filepath.Join(ExportDir, "day", day, code+".csv")
			if err = save(trades, output); err != nil {
				b.Logf("[ERR] [%s] %s\n", code, err)
				b.Flush()
				return
			}

		})
	}

	b.Wait()

	//压缩
	logs.Info("[导出] 今日数据压缩...")
	zipFilename := filepath.Join(ExportDir, "day", day+".zip")
	err := zip.Encode(
		filepath.Join(ExportDir, "day", day),
		zipFilename,
	)
	if err != nil {
		return err
	}

	//重命名
	logs.Info("[导出] 今日数据重命名...")
	<-time.After(time.Second * 5)
	err = os.Rename(zipFilename, filepath.Join(UploadDir, year, "每日数据", day+".zip"))
	if err != nil {
		return err
	}

	logs.Info("[导出] 今日数据完成...")

	return nil
}
