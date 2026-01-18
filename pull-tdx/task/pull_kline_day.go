package task

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/excel"
	"github.com/injoyai/goutil/str/bar"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"github.com/injoyai/tdx/protocol"
)

func NewPullKlineDay(codes []string, dir string) *PullKlineDay {
	return &PullKlineDay{
		Codes: codes,
		Dir:   dir,
	}
}

type PullKlineDay struct {
	Dir   string
	Codes []string
}

func (this *PullKlineDay) Name() string {
	return "拉取k线按天"
}

func (this *PullKlineDay) Run(ctx context.Context, m *tdx.Manage) error {
	return this.pullDayKline(ctx, m)
}

func (this *PullKlineDay) pullDayKline(ctx context.Context, m *tdx.Manage) error {
	codes := m.Codes.GetStockCodes()
	codes = append(codes, m.Codes.GetETFCodes()...)
	b := bar.New(int64(len(codes)))
	b.AddOption(func(f *bar.Format) {
		f.Entity.SetFormatter(func(e *bar.Format) string {
			return fmt.Sprintf("\r%s [%s] %s  %s  %s  %-10s",
				time.Now().Format(time.TimeOnly),
				this.Name(),
				e.Bar,
				e.RateSize,
				e.Speed,
				e.Used,
			)
		})
	})
	b.Add(0).Flush()

	data := [][]any{
		extend.DefaultDayKlineExportTitle,
		{"代码", "名称", "日期", "昨收", "开盘", "最高", "最低", "收盘", "成交量(股)", "成交额(元)", "涨跌(元)", "涨跌幅(%)", "换手率(%)", "流通股本(股)", "总股本(股)", "分红(元/股)", "配股价", "送转股", "配股"},
		//{"序号", "代码", "名称", "日期", "昨收", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "振幅", "涨跌幅"},
	}
	for i, code := range codes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		f := func() error {
			return m.Do(func(c *tdx.Client) (err error) {
				resp, err := c.GetKlineDay(code, 0, 2)
				if err == nil && len(resp.List) == 2 {
					var l *protocol.Kline
					switch len(resp.List) {
					case 1:
						l = resp.List[0]
					case 2:
						l = resp.List[1]
					default:
						return
					}

					x := []any{
						code,
						m.Codes.GetName(code),
						time.Now().Format(time.DateOnly),
						l.Last.Float64(),
						l.Open.Float64(),
						l.Close.Float64(),
						l.High.Float64(),
						l.Low.Float64(),
						l.Volume * 100,
						l.Amount.Float64(),
						l.RisePrice().Float64(),
						l.RiseRate(),
					}

					if eq := m.Gbbq.GetEquity(code, time.Now()); eq != nil {
						x = append(x, eq.Turnover(l.Volume*100), eq.Float, eq.Total)
					} else {
						x = append(x, "", "", "")
					}

					if xrxd := m.Gbbq.GetXRXDs(code); xrxd != nil {
						for _, v := range m.Gbbq.GetXRXDs(code) {
							if v.Time.Sub(v.Time) == 0 {
								x = append(x, v.Fenhong/10, v.Peigujia, v.Songzhuangu, v.Peigu)
								break
							}
						}
					}

					data = append(data, x)
				}

				return err
			})
		}
		err := g.Retry(f, tdx.DefaultRetry)
		logs.PrintErr(err)
		b.SetCurrentFlush(int64(i + 1))
	}
	buf, err := excel.ToCsv(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(this.Dir, "日线", time.Now().Format("2006/2006-01-02")+".csv")
	return oss.New(filename, buf)
}
