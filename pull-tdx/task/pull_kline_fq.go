package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"io"
	"net/http"
	"path/filepath"
	"pull-tdx/model"
	"strings"
	"time"
)

func NewPullKlineFQ(codes []string, exportDir string) *PullKlineFQ {
	return &PullKlineFQ{
		ExportDir: exportDir,
		Codes:     codes,
	}
}

type PullKlineFQ struct {
	ExportDir string
	Codes     []string //指定的代码
}

func (this *PullKlineFQ) Name() string {
	return "拉取复权日线"
}

func (this *PullKlineFQ) Run(ctx context.Context, m *tdx.Manage) error {
	t := &Range[string]{
		Codes:   GetCodes(m, this.Codes),
		Limit:   1,
		Retry:   DefaultRetry,
		Handler: this,
	}
	return t.Run(ctx, m)
}

func (this *PullKlineFQ) Handler(ctx context.Context, m *tdx.Manage, code string) (err error) {
	var resp *protocol.KlineResp
	err = m.Do(func(c *tdx.Client) error {
		resp, err = c.GetKlineDayAll(code)
		return err
	})
	if err != nil {
		return err
	}
	mAmount := make(map[int64]protocol.Price)
	for _, v := range resp.List {
		mAmount[v.Time.Unix()] = v.Amount
	}

	{
		ls, err := this.GetTHSDayKline(code, THS_QFQ)
		if err != nil {
			return err
		}
		filename := filepath.Join(this.ExportDir, "日线_前复权", code+".csv")
		err = this.save(filename, code, m.Codes.GetName(code), ls, mAmount)
		if err != nil {
			return err
		}
	}
	<-time.After(time.Millisecond * 20)
	{
		ls, err := this.GetTHSDayKline(code, THS_HFQ)
		if err != nil {
			return err
		}
		filename := filepath.Join(this.ExportDir, "日线_后复权", code+".csv")
		return this.save(filename, code, m.Codes.GetName(code), ls, mAmount)
	}

}

func (this *PullKlineFQ) save(filename string, code, name string, ls []*model.Kline, mAmount map[int64]protocol.Price) error {
	data := [][]any{
		{"日期", "时间", "代码", "名称", "开盘", "最高", "最低", "收盘", "总手", "金额"},
	}
	for i, v := range ls {
		t := time.Unix(v.Date, 0)
		data = append(data, []any{
			t.Format(time.DateOnly),
			t.Format("15:01"),
			code,
			name,
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			mAmount[ls[i].Date].Float64(),
		})
	}

	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}

const (
	UrlTHSDayKline       = "http://d.10jqka.com.cn/v6/line/hs_%s/0%d/all.js"
	THS_QFQ        uint8 = 1 //前复权
	THS_HFQ        uint8 = 2 //后复权
)

/*
GetTHSDayKline
前复权,和通达信对的上,和东方财富对不上
后复权,和通达信,东方财富都对不上
*/
func (this *PullKlineFQ) GetTHSDayKline(code string, _type uint8) ([]*model.Kline, error) {
	if _type != THS_QFQ && _type != THS_HFQ {
		return nil, fmt.Errorf("数据类型错误,例如:前复权1或后复权2")
	}

	code = protocol.AddPrefix(code)
	if len(code) != 8 {
		return nil, fmt.Errorf("股票代码错误,例如:SZ000001或000001")
	}

	u := fmt.Sprintf(UrlTHSDayKline, code[2:], _type)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	/*
	 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) '
	                      'Chrome/90.0.4430.212 Safari/537.36',
	        'Referer': 'http://stockpage.10jqka.com.cn/',
	        'DNT': '1',
	*/
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36 Edg/89.0.774.54")
	req.Header.Set("Referer", "http://stockpage.10jqka.com.cn/")
	req.Header.Set("DNT", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status %s", resp.Status)
	}

	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	n := bytes.IndexByte(bs, '(')
	if len(bs) > 1 {
		bs = bs[n+1 : len(bs)-1]
	}

	m := map[string]any{}
	err = json.Unmarshal(bs, &m)
	if err != nil {
		return nil, err
	}

	total := conv.Int(m["total"])
	sortYears := conv.Interfaces(m["sortYear"])
	priceFactor := conv.Float64(m["priceFactor"])
	prices := strings.Split(conv.String(m["price"]), ",")
	dates := strings.Split(conv.String(m["dates"]), ",")
	volumes := strings.Split(conv.String(m["volumn"]), ",")

	//好像到了22点,总数量会比实际多1
	if total == len(dates)+1 && total == len(volumes)+1 {
		total -= 1
	}
	//判断数量是否对应
	if total*4 != len(prices) || total != len(dates) || total != len(volumes) {
		return nil, fmt.Errorf("total=%d prices=%d dates=%d volumns=%d", total, len(prices), len(dates), len(volumes))
	}

	mYear := make(map[int][]string)
	index := 0
	for i, v := range sortYears {
		if ls := conv.Ints(v); len(ls) == 2 {
			year := conv.Int(ls[0])
			length := conv.Int(ls[1])
			if i == len(sortYears)-1 {
				mYear[year] = dates[index:]
				break
			}
			if len(dates) < index+length {
				logs.Err("意外的错误")
				mYear[year] = dates[index:]
				break
			}
			mYear[year] = dates[index : index+length]
			index += length
		}
	}

	ls := []*model.Kline(nil)
	i := 0
	nowYear := time.Now().Year()
	for year := 1990; year <= nowYear; year++ {
		for _, d := range mYear[year] {
			x, err := time.Parse("0102", d)
			if err != nil {
				return nil, err
			}
			x = time.Date(year, x.Month(), x.Day(), 15, 0, 0, 0, time.Local)
			low := protocol.Price(conv.Float64(prices[i*4+0]) * 1000 / priceFactor)
			ls = append(ls, &model.Kline{
				Code:   protocol.AddPrefix(code),
				Date:   x.Unix(),
				Open:   protocol.Price(conv.Float64(prices[i*4+1])*1000/priceFactor) + low,
				High:   protocol.Price(conv.Float64(prices[i*4+2])*1000/priceFactor) + low,
				Low:    low,
				Close:  protocol.Price(conv.Float64(prices[i*4+3])*1000/priceFactor) + low,
				Volume: (conv.Int64(volumes[i]) + 50) / 100,
			})
			i++
		}
	}

	return ls, nil
}
