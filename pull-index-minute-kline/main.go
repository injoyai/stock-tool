package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/goutil/str/bar/v2"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"os"
	"path/filepath"
	"time"
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
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
}

func main() {
	m, err := tdx.NewManage(&tdx.ManageConfig{Number: Clients})
	logs.PanicErr(err)

	c := cron.New(cron.WithSeconds())
	c.AddFunc(Spec, func() { Run(m) })

	if Startup {
		Run(m)
	}

	c.Run()
}

func Run(m *tdx.Manage) {
	Update(m)
	Export()
	logs.Info("更新完成...")
}

func Update(m *tdx.Manage) {

	b := bar.NewCoroutine(len(Codes), Coroutine)
	defer b.Close()

	for i := range Codes {
		code := Codes[i]
		b.Go(func() {
			b.SetPrefix("[更新][" + code + "]")
			b.Flush()
			err := m.Do(func(c *tdx.Client) error {
				return update(c, m.Workday, code)
			})
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err.Error())
				b.Flush()
			}
		})
	}

	b.Wait()

}

func Export() {
	b := bar.NewCoroutine(len(Codes), 3)
	defer b.Close()

	for i := range Codes {
		code := Codes[i]
		b.Go(func() {
			b.SetPrefix("[导出][" + code + "]")
			b.Flush()
			err := exportThisYear(code)
			if err != nil {
				b.Logf("[ERR] [%s] %s", code, err.Error())
				b.Flush()
			}
		})
	}

	b.Wait()
}

func update(c *tdx.Client, w *tdx.Workday, code string) error {
	dir := filepath.Join(DatabaseDir, conv.String(time.Now().Year()))
	filename := filepath.Join(dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	last := new(KlineMinute1)
	_, err = db.Desc("Date").Get(last)
	if err != nil {
		return err
	}

	if last.Date == 0 {
		last.Date = time.Now().AddDate(0, -4, 0).Unix()
	}

	ks := []*KlineBase(nil)
	w.Range(time.Unix(last.Date, 0).AddDate(0, 0, 1), time.Now(), func(t time.Time) bool {
		var resp *protocol.TradeResp
		resp, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		for _, v := range resp.List.Klines() {
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
		return true
	})
	if err != nil {
		return err
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

func pullToDB(c *tdx.Client, code string) error {

	return nil
}

func exportThisYear(code string) error {
	year := time.Now().Year()
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

func save(ks []*KlineBase, code, _type string, year int) error {
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
	filename := filepath.Join(ExportDir, conv.String(year), _type, code+".csv")
	if err = oss.New(filename, buf); err != nil {
		return err
	}
	<-time.After(time.Millisecond * 100)
	uploadFilename := filepath.Join(UploadDir, conv.String(year), _type, code+".csv")
	os.MkdirAll(filepath.Dir(uploadFilename), os.ModePerm)
	return os.Rename(filename, uploadFilename)
}
