package api

import (
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/tdx"
	"path/filepath"
	"strings"
	"time"
)

func Dial(log func(s string)) *Client {
	df := tdx.NewHostDial(nil)
	for {
		c, err := tdx.DialWith(
			df,
			tdx.WithDebug(),
			tdx.WithDebug(),
		)
		if err == nil {
			return &Client{Client: c}
		}
		if log != nil {
			log("连接服务失败: " + err.Error())
		}
		<-time.After(time.Second * 2)
	}
}

type Client struct {
	Codes []string
	Dir   string //保存数据的路径
	*tdx.Client
}

func (this *Client) DownloadTodayAll() error {

	if len(this.Codes) == 0 {
		this.Codes = []string{"sz000001"}
	}

	for _, code := range this.Codes {
		if err := this.DownloadToday(code); err != nil {
			return err
		}
	}
	return nil
}

func (this *Client) DownloadToday(code string) error {
	resp, err := this.GetMinuteTradeAll(code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"时间", "价格(分)", "成交量(手)", "成交额", "笔数", "方向", "均量", "均额", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				strings.ReplaceAll(v.Time, ":", ""),
				v.Price.Int64(),
				v.Volume,
				e,
				v.Number,
				getBuySell(v.Status),
				fmt.Sprintf("%.2f", float64(v.Volume)/float64(v.Number)),
				fmt.Sprintf("%.2f", float64(e)/float64(v.Number)),
				v.Price.Int64() * int64(v.Volume),
			},
		)
	}

	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}

	if time.Now().Hour() < 15 {
		code += "(不全)"
	}

	return oss.New(filepath.Join(this.Dir, time.Now().Format("20060102"), code+".csv"), buf)
}

func (this *Client) DownloadHistoryAll(start, end time.Time, log func(s string)) error {

	if len(this.Codes) == 0 {
		this.Codes = []string{"sz000001"}
	}

	for i, code := range this.Codes {
		for ; start.Unix() <= end.Unix(); start = start.Add(time.Hour * 24) {
			if err := this.DownloadHistory(start, code); err != nil {
				return err
			}
		}
		log(fmt.Sprintf("进度: %d/%d", i+1, len(this.Codes)))
	}

	return nil
}

func (this *Client) DownloadHistory(t time.Time, code string) error {
	date := t.Format("20060102")
	resp, err := this.GetHistoryMinuteTradeAll(date, code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"时间", "价格(分)", "成交量(手)", "成交额", "方向", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				strings.ReplaceAll(v.Time, ":", ""),
				v.Price.Int64(),
				v.Volume,
				e,
				getBuySell(v.Status),
				v.Price.Int64() * int64(v.Volume),
			},
		)
	}

	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}

	return oss.New(filepath.Join(this.Dir, date, code+".csv"), buf)
}

func getBuySell(n int) string {
	switch n {
	case 0:
		return "B"
	case 1:
		return "S"
	case 2:
		return "B"
	default:
		return conv.String(n)
	}
}
