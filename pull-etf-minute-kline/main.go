package main

import (
	"errors"
	"fmt"
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
	codes       = cfg.GetStrings("codes")
	startup     = cfg.GetBool("startup")
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.Info("版本:", "v1.1")
	logs.Info("详情:", "修复bug")
	fmt.Println("=====================================================")
	logs.Info("立即执行:", startup)
	logs.Info("代码地址:", address)
	logs.Info("连接数量:", clients)
	logs.Info("协程数量:", coroutines)
	logs.Info("定时规则:", spec)
	fmt.Println("=====================================================")
}

func main() {

	//初始化
	m, err := tdx.NewManage(
		tdx.WithClients(clients),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) { return extend.DialCodesHTTP(address) }),
	)
	logs.PanicErr(err)

	//是否立即更新
	if startup {
		logs.PanicErr(Update(m))
	}

	//定时更新
	err = m.AddWorkdayTask(spec, func(m *tdx.Manage) {
		logs.PrintErr(Update(m))
	})
	logs.PanicErr(err)

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
	if err := zip.Encode(filepath.Join(exportDir, year, "1分钟"), filepath.Join(uploadDir, year, "1分钟.zip")); err != nil {
		return err
	}
	if err := zip.Encode(filepath.Join(exportDir, year, "5分钟"), filepath.Join(uploadDir, year, "5分钟.zip")); err != nil {
		return err
	}
	if err := zip.Encode(filepath.Join(exportDir, year, "15分钟"), filepath.Join(uploadDir, year, "15分钟.zip")); err != nil {
		return err
	}
	if err := zip.Encode(filepath.Join(exportDir, year, "30分钟"), filepath.Join(uploadDir, year, "30分钟.zip")); err != nil {
		return err
	}
	if err := zip.Encode(filepath.Join(exportDir, year, "60分钟"), filepath.Join(uploadDir, year, "60分钟.zip")); err != nil {
		return err
	}
	return nil
}

func update(c *tdx.Client, year string, code string) error {

	//连接数据库
	filename := filepath.Join(databaseDir, year, code+".db")
	db, err := xorms.NewSqlite(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Sync2(new(KlineMinute1))
	if err != nil {
		return err
	}

	//读取数据库最后一条数据
	last := new(KlineMinute1)
	_, err = db.Desc("Date").Get(last)
	if err != nil {
		return err
	}

	//拉取数据
	resp, err := c.GetKlineMinuteUntil(code, func(k *protocol.Kline) bool { return k.Time.Unix() <= last.Date })
	if err != nil {
		return err
	}

	//更新到数据库
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err := session.Where("Date>=?", last.Date).Delete(new(KlineMinute1)); err != nil {
			return err
		}
		for _, v := range resp.List {
			if v.Time.Unix() >= last.Date {
				if _, err = session.Insert(&KlineMinute1{
					KlineBase: KlineBase{
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
						Volume: v.Volume,
						Amount: v.Amount.Float64(),
					},
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
	list := []*KlineMinute1(nil)
	if err = db.Asc("Date").Find(&list); err != nil {
		return err
	}

	ts := make(protocol.Klines, 0, len(list))
	for _, v := range list {
		ts = append(ts, &protocol.Kline{
			Open:   protocol.Price(v.Open * 1000),
			High:   protocol.Price(v.High * 1000),
			Low:    protocol.Price(v.Low * 1000),
			Close:  protocol.Price(v.Close * 1000),
			Order:  0,
			Volume: v.Volume,
			Amount: protocol.Price(v.Amount * 1000),
			Time:   time.Unix(v.Date, 0),
		})
	}

	if err = save(ts, "1分钟", year, code); err != nil {
		return err
	}

	if err = save(ts.Merge(5), "5分钟", year, code); err != nil {
		return err
	}

	if err = save(ts.Merge(15), "15分钟", year, code); err != nil {
		return err
	}

	if err = save(ts.Merge(30), "30分钟", year, code); err != nil {
		return err
	}

	if err = save(ts.Merge(60), "60分钟", year, code); err != nil {
		return err
	}

	return nil
}

func save(list protocol.Klines, _type, year, code string) error {
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
		data = append(data, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
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

	exportFilename := filepath.Join(exportDir, conv.String(year), _type, code+".csv")
	return oss.New(exportFilename, buf)
}
