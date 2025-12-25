package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"xorm.io/xorm"
)

var (
	DatabaseDir = "./data/database/kline"
	ExportDir   = "./data/export"
	UploadDir   = "./data/upload"
	Codes       = cfg.GetStrings("codes")
	Startup     = cfg.GetBool("startup", false)
	Clients     = cfg.GetInt("clients", 3)
	Coroutine   = cfg.GetInt("coroutine", 3)
	Spec        = cfg.GetString("spec", "0 1 15 * * *")
	Address     = cfg.GetString("address", "http://192.168.1.103:20000")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v1.3")
	logs.Info("详情:", "升级版本,优化版")
}

func main() {

	//初始化
	m, err := tdx.NewManage(
		tdx.WithClients(Clients),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) { return extend.DialCodesHTTP(Address) }),
	)
	logs.PanicErr(err)

	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(Spec, func() {
		if !m.Workday.TodayIs() {
			logs.Err("今天不是工作日")
			return
		}
		Run(m, Codes)
	})

	if Startup {
		Run(m, Codes)
	}

	cr.Run()
}

func Run(m *tdx.Manage, codes []string) {
	if len(codes) == 0 {
		codes = m.Codes.GetIndexCodes()
	}
	Update(m, codes)
	Export(codes)

	logs.Info("更新完成...")
}

func Update(m *tdx.Manage, codes []string) {

	b := bar.NewCoroutine(len(codes), Coroutine)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.Go(func() {
			b.SetPrefix("[更新][" + code + "]")
			b.Flush()
			err := m.Do(func(c *tdx.Client) error {
				return update(c, code)
			})
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err.Error())
				b.Flush()
			}
		})
	}

	b.Wait()

}

func Export(codes []string) {

	year := conv.String(time.Now().Year())

	os.MkdirAll(filepath.Join(ExportDir, year), os.ModePerm)
	os.MkdirAll(filepath.Join(UploadDir, year), os.ModePerm)

	b := bar.NewCoroutine(len(codes), 3)
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.Go(func() {
			b.SetPrefix("[导出][" + code + "]")
			b.Flush()
			err := export(year, code)
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err.Error())
				b.Flush()
			}
		})
	}

	b.Wait()

	for _, v := range []string{"1分钟", "5分钟", "15分钟", "30分钟", "60分钟"} {
		err := zip.Encode(
			filepath.Join(ExportDir, year, v),
			filepath.Join(ExportDir, year, v+".zip"),
		)
		logs.PrintErr(err)
		err = os.Rename(
			filepath.Join(ExportDir, year, v+".zip"),
			filepath.Join(UploadDir, year, v+".zip"),
		)
		logs.PrintErr(err)
	}

}

func update(c *tdx.Client, code string) error {
	now := time.Now()
	year := now.Year()
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	dir := filepath.Join(DatabaseDir, conv.String(year))
	filename := filepath.Join(dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Sync2(new(KlineMinute1)); err != nil {
		return err
	}

	last := new(KlineMinute1)
	_, err = db.Desc("Date").Get(last)
	if err != nil {
		return err
	}

	if last.Date == 0 {
		last.Date = now.AddDate(0, -4, 0).Unix()
		if last.Date < yearStart.Unix() {
			last.Date = yearStart.Unix()
		}
	}

	lastTime := time.Unix(last.Date, 0)
	ks := []*KlineBase(nil)
	resp, err := c.GetIndexUntil(protocol.TypeKlineMinute, code, func(k *protocol.Kline) bool {
		return k.Time.Before(lastTime)
	})
	if err != nil {
		return err
	}
	for _, v := range resp.List {
		if v.Time.After(lastTime) {
			ks = append(ks, &KlineBase{
				Date:   v.Time.Unix(),
				Year:   v.Time.Year(),
				Month:  int(v.Time.Month()),
				Day:    v.Time.Day(),
				Hour:   v.Time.Hour(),
				Minute: v.Time.Minute(),
				Open:   v.Open.Float64(),
				High:   v.High.Float64(),
				Low:    v.Low.Float64(),
				Close:  v.Close.Float64(),
				Volume: 0,
				Amount: float64(v.Volume * 100),
			})
		}
	}

	return db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range ks {
			_, err = session.Table(new(KlineMinute1)).Insert(v)
			if err != nil {
				return err
			}
		}
		return nil
	})

}

func export(year, code string) error {
	dir := filepath.Join(DatabaseDir, conv.String(year))
	filename := filepath.Join(dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	data := Klines{}
	err = db.Table(new(KlineMinute1)).Find(&data)
	if err != nil {
		return err
	}
	err = save(data, code, "1分钟", year)
	if err != nil {
		return err
	}
	err = save(data.Merge(5), code, "5分钟", year)
	if err != nil {
		return err
	}
	err = save(data.Merge(15), code, "15分钟", year)
	if err != nil {
		return err
	}
	err = save(data.Merge(30), code, "30分钟", year)
	if err != nil {
		return err
	}
	err = save(data.Merge(60), code, "60分钟", year)
	if err != nil {
		return err
	}
	return nil
}

func save(ks []*KlineBase, code, _type string, year string) error {
	data := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
	}
	for _, v := range ks {
		t := time.Unix(v.Date, 0)
		data = append(data, []any{
			t.Format(time.DateOnly),
			t.Format("15:04"),
			v.Open,
			v.High,
			v.Low,
			v.Close,
			v.Volume,
			v.Amount,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(ExportDir, year, _type, code+".csv")
	return oss.New(filename, buf)
}
