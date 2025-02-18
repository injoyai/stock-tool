package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
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
			tdx.WithRedial(),
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
	GetCodes func() ([]string, error)
	Dir      string //保存数据的路径
	*tdx.Client
}

func (this *Client) DownloadTodayAll(ctx context.Context, log func(s string), plan func(cu, to int)) error {

	codes, err := this.GetCodes()
	if err != nil {
		return err
	}

	if len(codes) == 0 {
		return errors.New("没有指定股票")
	}

	total := len(codes)
	plan(0, total)
	for i, code := range codes {
		select {
		case <-ctx.Done():
			return errors.New("手动停止")
		default:
		}
		err := this.DownloadToday(code)
		plan(i+1, total)
		if err != nil {
			log(fmt.Sprintf("代码: %s, 失败:%v", code, err))
			continue
		}
	}
	return nil
}

func (this *Client) DownloadToday(code string) (err error) {
	code, err = fullCode(code)
	if err != nil {
		return err
	}
	resp, err := this.GetMinuteTradeAll(code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"日期", "时间", "价格(分)", "成交量(手)", "成交额", "笔数", "方向", "均量", "均额", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				time.Now().Format("2006/01/02"),
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

	code = strings.TrimPrefix(code, "sz")
	code = strings.TrimPrefix(code, "sh")

	return oss.New(filepath.Join(this.Dir, time.Now().Format("2006-01-02"), code+".csv"), buf)
}

func (this *Client) DownloadHistoryAll(ctx context.Context, start, end time.Time, log func(s string), plan func(cu, to int)) error {

	codes, err := this.GetCodes()
	if err != nil {
		return err
	}

	if len(codes) == 0 {
		return errors.New("没有指定股票")
	}

	total := len(codes)
	plan(0, total)
	for i, code := range codes {
		select {
		case <-ctx.Done():
			return errors.New("手动停止")
		default:
		}
		for date := start; date.Unix() <= end.Unix(); date = date.Add(time.Hour * 24) {
			select {
			case <-ctx.Done():
				return errors.New("手动停止")
			default:
			}
			err := this.DownloadHistory(date, code)
			if err != nil {
				log(fmt.Sprintf("日期: %s, 代码: %s, 失败:%v", date.Format("2006/01/02"), code, err))
				break
			}
		}
		plan(i+1, total)
	}

	return nil
}

func (this *Client) DownloadHistory(t time.Time, code string) (err error) {
	defer g.Recover(&err)
	code, err = fullCode(code)
	if err != nil {
		return err
	}
	resp, err := this.GetHistoryMinuteTradeAll(t.Format("20060102"), code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"日期", "时间", "价格(分)", "成交量(手)", "成交额", "笔数", "方向", "均量", "均额", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				t.Format("2006/01/02"),
				strings.ReplaceAll(v.Time, ":", ""),
				v.Price.Int64(),
				v.Volume,
				"",
				e,
				"",
				"",
				getBuySell(v.Status),
				v.Price.Int64() * int64(v.Volume),
			},
		)
	}

	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}

	code = strings.TrimPrefix(code, "sz")
	code = strings.TrimPrefix(code, "sh")

	return oss.New(filepath.Join(this.Dir, t.Format("2006-01-02"), code[2:]+".csv"), buf)
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

func fullCode(code string) (string, error) {
	code = strings.ToLower(code)
	if len(code) == 6 {
		switch {
		case strings.HasPrefix(code, "0"):
			return "sz" + code, nil
		case strings.HasPrefix(code, "30"):
			return "sz" + code, nil
		case strings.HasPrefix(code, "6"):
			return "sh" + code, nil
		}
	} else if len(code) == 8 {
		switch {
		case strings.HasPrefix(code, "sh"):
			return code, nil
		case strings.HasPrefix(code, "sz"):
			return code, nil
		}
	}
	return "", errors.New("无效代码: " + code)
}
