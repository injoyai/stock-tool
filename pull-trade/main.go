package main

import (
	"context"
	"fmt"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	StartDate    = time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local)
	DefaultRetry = 3
)

var (
	Clients     int
	Coroutines  int
	Tasks       int
	DatabaseDir string
	ExportDir   string
	Spec        string
	Codes       []string
	Startup     bool
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v0.2.2")
	logs.Info("说明:", "增加定时任务")
	fmt.Println("=====================================================")
	initCfg("./config/config.yaml")
}

func initCfg(filename string) {
	cfg.Init(cfg.WithFile(filename))
	Clients = cfg.GetInt("clients", 4)
	Coroutines = cfg.GetInt("coroutines", 10)
	Tasks = cfg.GetInt("tasks", 2)
	DatabaseDir = cfg.GetString("database", "./data/database")
	ExportDir = cfg.GetString("export", "./data/export")
	Spec = cfg.GetString("spec", "0 1 19 * * *")
	Codes = cfg.GetStrings("codes")
	Startup = cfg.GetBool("startup")
}

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	p := NewPullKlineDay(
		Codes,
		"./data/klineday",
	)

	err = p.pullDay(
		m, Codes,
		time.Date(2025, 2, 12, 0, 0, 0, 0, time.Local),
		time.Date(2025, 7, 8, 0, 0, 0, 0, time.Local),
	)

	//err = p.Run(context.Background(), m)
	logs.PrintErr(err)

	//err = exportKline(m)
	//logs.PrintErr(err)
	//g.Input("结束...")
	//select {}
}

/*









 */

func updateAndExport() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)
	corn := cron.New(cron.WithSeconds())
	corn.AddFunc("0 10 15 * * *", func() {
		logs.Info("更新数据...")
		err = update(m)
		logs.PrintErr(err)
		logs.Info("导出数据...")
		err = exportKline(m)
		logs.PrintErr(err)
		logs.Info("任务完成...")
	})
	corn.Run()
}

func exportKline(m *tdx.Manage) error {
	e := NewExportKline(
		Codes,
		[]int{2023, 2024, 2025},
		filepath.Join(DatabaseDir, "kline"),
		filepath.Join(ExportDir),
	)
	return e.Run(context.Background(), m)
}

func update(m *tdx.Manage) error {
	u := NewUpdateKline(Codes, filepath.Join(DatabaseDir, "kline"), Coroutines)
	return u.Run(context.Background(), m)
}

func invalidFolder() {
	oss.RangeFileInfo("./data/database/trade/", func(info *oss.FileInfo) (bool, error) {
		if info.IsDir() {
			fs, err := oss.ReadFileInfos(info.FullName())
			if err != nil {
				return false, err
			}
			has := false
			for _, f := range fs {
				if strings.Contains(f.Name(), "-2025.db") {
					has = true
				}
			}
			if !has {
				logs.Debug("无效数据:", info.FullName())
			}
		}
		return true, nil
	})
}

func clear() {
	t := time.Date(2025, 6, 30, 0, 0, 0, 0, time.Local)
	oss.RangeFileInfo("./data/database/trade/", func(info *oss.FileInfo) (bool, error) {
		if info.IsDir() {
			oss.RangeFileInfo(info.FullName(), func(info *oss.FileInfo) (bool, error) {
				if info.Size() == 0 {
					if info.ModTime().After(t) {
						logs.Debug("删除:", info.FullName(), info.ModTime().Format(time.DateTime))
						os.Remove(info.FullName())
						return true, nil
					} else {
						logs.Debug("保留:", info.FullName())
					}
				}
				return true, nil
			})
		}
		return true, nil
	})
}

func xxx() []string {
	//c, err := tdx.DialCodes("")
	//logs.PanicErr(err)
	//
	//codes := c.GetStocks()
	m := make(map[string]struct{})
	//for _, code := range codes {
	//	m[code] = struct{}{}
	//}

	err := oss.RangeFileInfo("./data/database/trade/", func(info *oss.FileInfo) (bool, error) {
		//delete(m, strings.Split(info.Name(), ".")[0])
		m[strings.Split(info.Name(), ".")[0]] = struct{}{}
		return true, nil
	})
	logs.PanicErr(err)

	err = oss.RangeFileInfo("./data/database/kline/", func(info *oss.FileInfo) (bool, error) {
		delete(m, strings.Split(info.Name(), ".")[0])
		return true, nil
	})
	logs.PanicErr(err)

	ls := []string(nil)
	for k, _ := range m {
		ls = append(ls, k)
	}

	logs.Debug("数量:", len(ls))

	return ls
}

func convert() {
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)
	c := NewConvert(
		Codes,
		"",
		filepath.Join(DatabaseDir, "trade"),
		filepath.Join(DatabaseDir, "kline_append1"),
		filepath.Join(DatabaseDir, "kline_append2"),
		filepath.Join(DatabaseDir, "kline"),
		time.Date(2025, 5, 21, 0, 0, 0, 0, time.Local),
	)
	c.Run(context.Background(), m)
}

func export() {
	initCfg("./config/export.yaml")
	m, err := tdx.NewManage(nil)
	logs.PanicErr(err)
	e := NewExport(
		[]string{}, //"sz000001"},
		[]int{2015, 2016, 2017, 2018, 2019, 2020, 2021, 2022, 2023, 2024, 2025},
		filepath.Join(DatabaseDir, "trade"),
		ExportDir,
	)
	e.Run(context.Background(), m)
}

func pull() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	s := NewSqlite(
		Codes,
		filepath.Join(DatabaseDir, "trade"),
		Coroutines,
		Tasks,
	)

	t := cron.New(cron.WithSeconds())
	t.AddFunc(Spec, func() { s.Run(context.Background(), m) })
	if Startup {
		s.Run(context.Background(), m)
	}
	t.Run()
}
