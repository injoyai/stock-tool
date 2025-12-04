package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/compress/zip"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/injoyai/tdx/lib/xorms"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

var (
	clients     = cfg.GetInt("clients", 3)
	coroutines  = cfg.GetInt("coroutines", 10)
	retry       = cfg.GetInt("retry", 3)
	address     = cfg.GetString("address", "http://127.0.0.1:20000")
	spec        = cfg.GetString("spec", "0 15 15 * * *")
	databaseDir = cfg.GetString("database_dir", "./data/database/kline")
	exportDir   = cfg.GetString("export_dir", "./data/export")
	uploadDir   = cfg.GetString("upload_dir", "./data/upload")
	codes       = []string{
		"sz159399",
	}
)

func main() {

	m, err := tdx.NewManage(
		tdx.WithClients(clients),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) { return extend.DialCodesHTTP(address) }),
	)
	logs.PanicErr(err)

	//更新
	logs.PanicErr(Update(m))

	m.AddWorkdayTask(spec, func(m *tdx.Manage) {
		logs.PrintErr(Update(m))
	})

	select {}
}

func Update(m *tdx.Manage) error {
	defer func() { logs.Info("任务完成...") }()

	if len(codes) == 0 {
		codes = m.Codes.GetETFCodes()
	}

	b := bar.NewCoroutine(len(codes), coroutines)
	defer b.Close()

	year := conv.String(time.Now().Year())

	os.MkdirAll(filepath.Join(exportDir, year), os.ModePerm)
	os.MkdirAll(filepath.Join(uploadDir, year), os.ModePerm)

	for i := range codes {
		code := codes[i]
		b.GoRetry(func() error {
			if err := m.Do(func(c *tdx.Client) error { return update(c, year, code) }); err != nil {
				b.Logf("[错误] %s", err)
				b.Flush()
				return err
			}
			return export(year, code)
		}, retry)
	}

	b.Wait()

	logs.Info("开始压缩...")
	return zip.Encode(
		filepath.Join(exportDir, year),
		filepath.Join(uploadDir, year, year+".zip"),
	)
}

func update(c *tdx.Client, year string, code string) error {

	//连接数据库
	filename := filepath.Join(databaseDir, year, code+".db")
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Sync2(new(extend.Kline))
	if err != nil {
		return err
	}

	//读取数据库最后一条数据
	last := new(extend.Kline)
	_, err = db.Desc("Date").Get(last)
	if err != nil {
		return err
	}

	logs.Debug(time.Unix(last.Date, 0))

	//拉取数据
	resp, err := c.GetKlineMinuteUntil(code, func(k *protocol.Kline) bool { return k.Time.Unix() <= last.Date })
	if err != nil {
		return err
	}

	logs.Debug(len(resp.List))

	//更新到数据库
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err := session.Where("Date>=?", last.Date).Delete(new(extend.Kline)); err != nil {
			return err
		}
		for _, v := range resp.List {
			if v.Time.Unix() >= last.Date {
				if _, err = session.Insert(&extend.Kline{
					Date:   v.Time.Unix(),
					Open:   v.Open,
					High:   v.High,
					Low:    v.Low,
					Close:  v.Close,
					Volume: v.Volume,
					Amount: v.Amount,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})

}

// export 导出数据
func export(year string, code string) error {

	filename := filepath.Join(databaseDir, year, code+".db")
	if !oss.Exists(filename) {
		return errors.New("文件不存在: " + filename)
	}

	//打开数据库
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	//读取今年数据
	list := []*extend.Kline(nil)
	if err = db.Asc("Date").Find(&list); err != nil {
		return err
	}

	//导出
	data := [][]any{{
		"日期",
		"时间",
		"开盘",
		"最高",
		"最低",
		"收盘",
		"成交量(手)",
		"成交额(元)",
	}}

	for _, v := range list {
		t := time.Unix(v.Date, 0)
		data = append(data, []any{
			t.Format(time.DateOnly),
			t.Format(time.TimeOnly),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
		})
	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}

	exportFilename := filepath.Join(exportDir, conv.String(year), code+".csv")
	return oss.New(exportFilename, buf)
}
