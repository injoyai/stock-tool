package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg/v2"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"strings"
	"time"
)

var (
	Invalid  string                //有效期
	Codes    = map[string]string{} //拉取的指数代码
	Types    []string              //拉取的指数类型
	Hosts    []string              //服务器地址
	Filename string                //文件名
)

func init() {
	logs.SetFormatter(logs.TimeFormatter)
	logs.SetShowColor(false)

	ls := cfg.GetInterfaces("codes", []interface{}{
		map[string]string{"sh000001": "上证指数"},
		map[string]string{"sz399001": "深证成指"},
		map[string]string{"sh000016": "上证50"},
		map[string]string{"sh000010": "上证180"},
		map[string]string{"sh000300": "上证300"},
		map[string]string{"sh000905": "中证500"},
		map[string]string{"sh000852": "中证1000"},
		map[string]string{"sz399006": "创业板指"},
	})
	for _, v := range ls {
		if m, ok := v.(map[string]interface{}); ok {
			for k, v := range m {
				Codes[k] = conv.String(v)
			}
		}
	}

	Types = cfg.GetStrings("types", []string{"日", "周", "月", "季", "年"})

	Hosts = cfg.GetStrings("hosts", tdx.Hosts)

	Filename = cfg.GetString("filename", "./data/{type}线/{code}({name}).csv")

}

func main() {

	defer func() {
		g.Input("按回车键结束...")
	}()

	if len(Invalid) > 0 {
		t, err := time.Parse("2006-01-02", Invalid)
		if err != nil {
			logs.Err(err)
			return
		}
		if time.Now().After(t) {
			logs.Err("已过有效期: " + Invalid)
			return
		}
	}

	var c *tdx.Client
	var err error
	for _, host := range Hosts {
		c, err = tdx.Dial(host)
		if err != nil {
			logs.Err(err)
			return
		}
		logs.Infof("连接服务器[%s]成功...\n", host)
		break
	}
	if c == nil {
		return
	}
	c.Wait.SetTimeout(time.Second * 5)

	for _, _type := range Types {
		switch _type {
		case "日":
			err = do(c.GetKlineDayAll, _type, Filename)
			logs.PrintErr(err)

		case "周":
			err = do(c.GetKlineWeekAll, _type, Filename)
			logs.PrintErr(err)

		case "月":
			err = do(c.GetKlineMonthAll, _type, Filename)
			logs.PrintErr(err)

		case "季":
			err = do(c.GetKlineQuarterAll, _type, Filename)
			logs.PrintErr(err)

		case "年":
			err = do(c.GetKlineYearAll, _type, Filename)
			logs.PrintErr(err)

		}
	}

	logs.Info("拉取完成")

}

func do(f func(code string) (*protocol.KlineResp, error), _type, filename string) error {
	for code, name := range Codes {

		logs.Infof("开始拉取%s线: %s(%s)\n", _type, code, name)
		resp, err := f(code)
		if err != nil {
			logs.Err(err)
			continue
		}

		data := [][]any{
			{"日期", "开盘", "最高", "最低", "收盘", "成交量", "成交额", "涨跌价", "涨跌幅"},
		}
		for _, v := range resp.List {
			data = append(data, []any{v.Time.Format("2006-01-02"), v.Open.Float64(), v.High.Float64(), v.Low.Float64(), v.Close.Float64(), v.Volume, v.Amount.Float64(), v.RisePrice().Float64(), v.RiseRate()})
		}
		buf, err := excel.ToCsv(data)
		if err != nil {
			logs.Err(err)
			continue
		}

		_filename := strings.ReplaceAll(filename, "{type}", _type)
		_filename = strings.ReplaceAll(_filename, "{code}", code)
		_filename = strings.ReplaceAll(_filename, "{name}", name)
		oss.New(_filename, buf)
	}
	return nil
}
