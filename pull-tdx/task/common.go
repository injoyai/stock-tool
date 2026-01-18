package task

import (
	"pull-tdx/model"
	"time"

	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

var (
	title     = []any{"日期", "代码", "名称", "昨收", "开盘", "最高", "最低", "收盘", "成交量(股)", "成交额(元)", "涨跌(元)", "涨跌幅(%)"}
	titleMore = []any{"日期", "代码", "名称", "昨收", "开盘", "最高", "最低", "收盘", "成交量(股)", "成交额(元)", "涨跌(元)", "涨跌幅(%)",
		"换手率(%)", "流通股本(股)", "总股本(股)", "前复权因子", "后复权因子", "分红(元/股)", "配股价", "送转股", "配股"}
)

func klineToCsv2(code string, ks model.Klines, filename string, getName func(code string) string) error {
	ls := [][]any{title}
	for _, v := range ks {
		t := time.Unix(v.Date, 0)
		ls = append(ls, []any{
			t.Format(time.DateOnly),
			code,
			getName(code),
			v.Last.Float64(),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume * 100,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
		})
	}
	buf, err := excel.ToCsv(ls)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}

func dayKlineToCsv(gb tdx.IGbbq, code string, ls model.Klines, filename string, getName func(code string) string) error {

	ks := make(protocol.Klines, 0, len(ls))
	for _, v := range ls {
		ks = append(ks, &protocol.Kline{
			Last:      v.Last,
			Open:      v.Open,
			High:      v.High,
			Low:       v.Low,
			Close:     v.Close,
			Order:     0,
			Volume:    v.Volume,
			Amount:    v.Amount,
			Time:      time.Unix(v.Date, 0),
			UpCount:   0,
			DownCount: 0,
		})
	}

	fs := gb.GetFactors(code, ks)
	mFactor := map[string]*protocol.Factor{}
	for _, v := range fs {
		mFactor[v.Time.Format(time.DateOnly)] = v
	}

	xrxds := gb.GetXRXDs(code)
	mXrxd := map[string]*protocol.XRXD{}
	for _, v := range xrxds {
		mXrxd[v.Time.Format(time.DateOnly)] = v
	}

	data := [][]any{titleMore}
	for _, v := range ks {
		x := []any{
			v.Time.Format(time.DateOnly),
			code,
			getName(code),
			v.Last.Float64(),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume * 100,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
		}

		if eq := gb.GetEquity(code, v.Time); eq != nil {
			x = append(x, eq.Turnover(v.Volume*100), int64(eq.Float), int64(eq.Total))
		} else {
			x = append(x, "", "", "")
		}

		if f := mFactor[v.Time.Format(time.DateOnly)]; f != nil {
			x = append(x, f.QFQ, f.HFQ)
		} else {
			x = append(x, "", "")
		}

		if xr := mXrxd[v.Time.Format(time.DateOnly)]; xr != nil {
			x = append(x, xr.Fenhong, xr.Peigujia, xr.Songzhuangu, xr.Peigu)
		}

		data = append(data, x)
	}
	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}

func klineToCsv(code string, ks []*protocol.Kline, filename string, getName func(code string) string) error {
	ls := [][]any{title}
	for _, v := range ks {
		ls = append(ls, []any{
			v.Time.Format(time.DateOnly),
			code,
			getName(code),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume * 100,
			v.Amount.Float64(),
			v.RisePrice().Float64(),
			v.RiseRate(),
		})
	}
	buf, err := excel.ToCsv(ls)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}

func GetCodes(m *tdx.Manage, codes []string) []string {
	if len(codes) == 0 {
		return m.Codes.GetStockCodes()
	}
	return codes
}
