package task

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"
	"time"
)

func NewExportTrade(codes []string, databaseDir, exportDir string, limit int) *ExportTrade {
	return &ExportTrade{
		Codes:       codes,
		DatabaseDir: tradeDir(databaseDir),
		ExportDir:   exportDir,
		Limit:       limit,
	}
}

type ExportTrade struct {
	Codes       []string
	DatabaseDir tradeDir
	ExportDir   string
	Limit       int
}

func (this *ExportTrade) Name() string {
	return "导出分时成交数据"
}

func (this *ExportTrade) Run(ctx context.Context, m *tdx.Manage) error {
	r := &Range[string]{
		Codes:   GetCodes(m, this.Codes),
		Append:  nil,
		Limit:   this.Limit,
		Retry:   DefaultRetry,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *ExportTrade) Handler(ctx context.Context, m *tdx.Manage, code string) error {

	//取最新的年份进行导出
	year, filename := this.DatabaseDir.lastYear(code)
	b, err := db.Open(filename)
	if err != nil {
		return err
	}
	defer b.Close()

	all := model.Trades{}
	err = b.Find(&all)
	if err != nil {
		return err
	}

	return this.export(code, m.Codes.GetName(code), year, all)
}

func (this *ExportTrade) export(code, name string, year int, tss model.Trades) (err error) {

	kss1 := model.Klines(nil)
	kss5 := model.Klines(nil)
	kss15 := model.Klines(nil)
	kss30 := model.Klines(nil)
	kss60 := model.Klines(nil)

	//转成分时K线
	ks, err := tss.Klines1()
	if err != nil {
		return err
	}

	kss1 = append(kss1, ks...)
	kss5 = append(kss5, ks.Merge(5)...)
	kss15 = append(kss5, ks.Merge(15)...)
	kss30 = append(kss5, ks.Merge(30)...)
	kss60 = append(kss5, ks.Merge(60)...)

	filename := filepath.Join(this.ExportDir, "分时成交", code+"-"+conv.String(year)+".csv")
	filename1 := filepath.Join(this.ExportDir, "1分钟", code+"-"+conv.String(year)+".csv")
	filename5 := filepath.Join(this.ExportDir, "5分钟", code+"-"+conv.String(year)+".csv")
	filename15 := filepath.Join(this.ExportDir, "15分钟", code+"-"+conv.String(year)+".csv")
	filename30 := filepath.Join(this.ExportDir, "30分钟", code+"-"+conv.String(year)+".csv")
	filename60 := filepath.Join(this.ExportDir, "60分钟", code+"-"+conv.String(year)+".csv")

	err = this.exportTrade(filename, tss)
	if err != nil {
		return err
	}

	err = this.exportKline(filename1, code, name, kss1)
	if err != nil {
		return err
	}

	err = this.exportKline(filename5, code, name, kss5)
	if err != nil {
		return err
	}

	err = this.exportKline(filename15, code, name, kss15)
	if err != nil {
		return err
	}

	err = this.exportKline(filename30, code, name, kss30)
	if err != nil {
		return err
	}

	err = this.exportKline(filename60, code, name, kss60)
	if err != nil {
		return err
	}

	return nil
}

func (this *ExportTrade) exportKline(filename string, code, name string, ks model.Klines) error {
	logs.Debug(filename)
	data := [][]any{{"日期", "时间", "代码", "名称", "开盘", "最高", "最低", "收盘", "总手", "金额"}}
	for _, v := range ks {
		t := time.Unix(v.Date, 0)
		data = append(data, []any{
			t.Format("20060102"),
			t.Format("15:04"),
			code,
			name,
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

	return oss.New(filename, buf)
}

func (this *ExportTrade) exportTrade(filename string, ts model.Trades) error {
	data := [][]any{{"日期", "时间", "价格", "成交量(手)", "成交额", "方向(0买,1卖)"}}
	for _, v := range ts {
		t := v.ToTime()
		data = append(data, []any{
			t.Format(time.DateOnly),
			t.Format("15:04"),
			v.Price.Float64(),
			v.Volume,
			v.Amount().Float64(),
			v.Status,
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	return oss.New(filename, buf)
}
