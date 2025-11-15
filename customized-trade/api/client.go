package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/tdx"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func Dial(clients, disks int, timeout time.Duration, log func(s string)) *Client {
	if clients <= 0 {
		clients = 1
	}
	if disks <= 0 {
		disks = 1
	}
	if timeout <= time.Second {
		timeout = 2 * time.Second
	}

	df := tdx.NewHostDial(nil)
	for {
		p, err := NewPool(func() (*tdx.Client, error) {
			c, err := tdx.DialWith(df, tdx.WithRedial())
			if err != nil {
				return nil, err
			}
			c.Wait.SetTimeout(timeout)
			return c, nil
		}, clients)
		if err == nil {
			cli := &Client{
				Pool: p,
				Ch:   make(chan func(), disks),
			}
			go cli.Run(disks)
			return cli
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
	Pool     *Pool
	Ch       chan func()
}

func (this *Client) DownloadTodayAll(ctx context.Context, log func(s string), plan func(cu, to int)) error {

	c, err := this.Pool.Get()
	if err != nil {
		return err
	}
	defer this.Pool.Put(c)

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
		err := this.DownloadToday(c, code, log)
		plan(i+1, total)
		if err != nil {
			log(fmt.Sprintf("代码: %s, 失败:%v", code, err))
			continue
		}
	}
	return nil
}

func (this *Client) DownloadTodayAll2(ctx context.Context, log func(s string), plan func(cu, to int), dealErr func(code string, err error)) error {

	codes, err := this.GetCodes()
	if err != nil {
		return err
	}

	if len(codes) == 0 {
		return errors.New("没有指定股票")
	}

	total := len(codes)
	plan(0, total)
	for i := range codes {
		select {
		case <-ctx.Done():
			return errors.New("手动停止")

		default:
			code := codes[i]
			this.Pool.Go(func(c *tdx.Client) {
				err := this.DownloadToday(c, code, log)
				plan(i+1, total)
				if err != nil {
					log(fmt.Sprintf("代码: %s, 失败:%v", code, err))
					dealErr(code, err)
				}
			})

		}

	}
	return nil
}

func (this *Client) DownloadToday(c *tdx.Client, code string, log func(s string)) (err error) {

	code, err = fullCode(code)
	if err != nil {
		return err
	}
	resp, err := c.GetMinuteTradeAll(code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"日期", "时间", "价格", "成交量", "成交额", "笔数", "方向", "均量", "均额", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				time.Now().Format("2006/01/02"),
				v.Time.Format("1504"),
				v.Price.Int64(),
				v.Volume,
				e,
				v.Number,
				getBuySell(v.Time.Format("1504"), v.Status),
				fmt.Sprintf("%.2f", float64(v.Volume)/float64(v.Number)),
				fmt.Sprintf("%.2f", float64(e)/float64(v.Number)),
				v.Price.Int64() * int64(v.Volume),
			},
		)
	}

	this.Ch <- func() {
		buf, err := excel.ToCsv(data)
		if err != nil {
			log(err.Error())
			return
		}
		//if time.Now().Hour() < 15 {
		//	code += "(不全)"
		//}
		code = strings.TrimPrefix(code, "sz")
		code = strings.TrimPrefix(code, "sh")
		err = oss.New(filepath.Join(this.Dir, time.Now().Format("2006-01-02"), code+".csv"), buf)
		if err != nil {
			log(err.Error())
			return
		}
	}

	return nil

	//buf, err := excel.ToCsv(data)
	//if err != nil {
	//	return err
	//}
	//
	//if time.Now().Hour() < 15 {
	//	code += "(不全)"
	//}
	//
	//code = strings.TrimPrefix(code, "sz")
	//code = strings.TrimPrefix(code, "sh")
	//
	//return oss.New(filepath.Join(this.Dir, time.Now().Format("2006-01-02"), code+".csv"), buf)
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
	var index uint32
	for i := range codes {
		select {
		case <-ctx.Done():
			return errors.New("手动停止")

		default:

			code := codes[i]
			this.Pool.Go(func(c *tdx.Client) {
				for date := start; date.Unix() <= end.Unix(); date = date.Add(time.Hour * 24) {
					select {
					case <-ctx.Done():
						break

					default:
						err := this.DownloadHistory(c, date, code, log)
						if err != nil {
							log(fmt.Sprintf("日期: %s, 代码: %s, 失败:%v", date.Format("2006/01/02"), code, err))
							break
						}

					}
				}
				plan(int(atomic.AddUint32(&index, 1)), total)
			})
		}

	}

	return nil
}

func (this *Client) DownloadHistory(c *tdx.Client, t time.Time, code string, log func(s string)) (err error) {
	defer g.Recover(&err)
	code, err = fullCode(code)
	if err != nil {
		return err
	}
	resp, err := c.GetHistoryTradeDay(t.Format("20060102"), code)
	if err != nil {
		return err
	}

	data := [][]any{
		{"日期", "时间", "价格", "成交量", "成交额", "笔数", "方向", "均量", "均额", "成交额2"},
	}
	for _, v := range resp.List {
		//成交额
		e := (v.Price.Int64()*int64(v.Volume) + 500) / 1000
		data = append(data,
			[]any{
				t.Format("2006/01/02"),
				v.Time.Format("1504"),
				v.Price.Int64(),
				v.Volume,
				e,
				"",
				getBuySell(v.Time.Format("1504"), v.Status),
				"",
				"",
				v.Price.Int64() * int64(v.Volume),
			},
		)
	}

	this.Ch <- func() {
		buf, err := excel.ToCsv(data)
		if err != nil {
			log(err.Error())
			return
		}

		code = strings.TrimPrefix(code, "sz")
		code = strings.TrimPrefix(code, "sh")
		err = oss.New(filepath.Join(this.Dir, t.Format("2006-01-02"), code+".csv"), buf)
		if err != nil {
			log(err.Error())
			return
		}
	}

	return nil

	//buf, err := excel.ToCsv(data)
	//if err != nil {
	//	return err
	//}
	//
	//code = strings.TrimPrefix(code, "sz")
	//code = strings.TrimPrefix(code, "sh")
	//
	//return oss.New(filepath.Join(this.Dir, t.Format("2006-01-02"), code+".csv"), buf)
}

func (this *Client) Run(limit int) {
	c := chans.NewLimit(limit)
	for {
		select {
		case fn := <-this.Ch:
			c.Add()
			go func() {
				defer c.Done()
				fn()
			}()
		}
	}
}

func getBuySell(time string, n int) string {
	if time == "15:00" {
		return ""
	}
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
