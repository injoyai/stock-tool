package main

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
	"log"
	"path/filepath"
	"time"
)

var (
	testCodes = []string{
		"sz001914",
	}
	debug = len(testCodes) > 0
)

func main() {

	s := &Strategy{
		WindowsSize: 8,
		DayStart:    0,
		DayNumber:   100,
		Dir:         filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
	}

	if len(testCodes) == 0 {
		testCodes = tdx.DefaultCodes.GetStocks()
	}

	result := s.Find(testCodes)

	logs.Debug(result)

}

type Strategy struct {
	WindowsSize int    //窗口大小
	DayStart    uint16 //股票起始
	DayNumber   int    //股票天数
	Dir         string
}

func (this *Strategy) GetKlines(code string, limit int) (Klines, error) {
	data := []*extend.Kline(nil)
	db, err := sqlite.NewXorm(filepath.Join(this.Dir, code+".db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.Table(extend.NewKlineTable("DayKline", nil)).Desc("Date").Limit(limit).Find(&data)
	if err != nil {
		return nil, err
	}
	klines := Klines(nil)
	for i := len(data) - 1; i >= 0; i-- {
		v := data[i]
		klines = append(klines, &Kline{
			Time:  time.Unix(v.Date, 0),
			Open:  v.Open,
			High:  v.High,
			Low:   v.Low,
			Close: v.Close,
		})
	}
	return klines, nil
}

func (this *Strategy) Find(codes []string) []string {

	result := []string(nil)

	for _, code := range codes {

		ls, err := this.GetKlines(code, this.DayNumber)
		logs.PanicErr(err)

		h, l := ls.Vertexes(this.WindowsSize)

		if debug {
			for _, v := range h {
				logs.Debug(v.Kline)
			}
			for _, v := range l {
				logs.Debug(v.Kline)
			}
		}

		if Check(h, l, this.WindowsSize) {
			result = append(result, code)
			logs.Debug(code)
		}

	}

	return result
}

/*

一个底部(l1),一个顶部(h1),一个底部(l2),一个顶部(h2)

l2>l1 && h2>h1


*/

func Check(highs, lows []*Vertex, windowSize int) bool {
	if len(highs) < 2 || len(lows) < 2 {
		return false
	}

	//各取2个最新的顶部和底部
	h := highs[len(highs)-2:]
	l := lows[len(lows)-2:]

	if debug {
		log.Println(l[0].Kline)
		log.Println(h[0].Kline)
		log.Println(l[1].Kline)
		log.Println(h[1].Kline)
	}

	//判断顶点是否过远
	if int(time.Now().Sub(h[1].Kline.Time).Hours()/24) > windowSize*2 {
		return false
	}

	//判断时间是否交替
	if !(h[1].Kline.Time.After(l[1].Kline.Time) &&
		l[1].Kline.Time.After(h[0].Kline.Time) &&
		h[0].Kline.Time.After(l[0].Kline.Time)) {
		return false
	}

	//判断间隔是否过近
	if h[1].Index-l[1].Index < windowSize || l[1].Index-h[0].Index < windowSize || h[0].Index-l[0].Index < windowSize {
		return false
	}

	//判断顶部和底部逐步上升
	if h[1].Kline.High <= h[0].Kline.High || l[1].Kline.Low <= l[0].Kline.Low {
		return false
	}

	//底部不能大于顶部
	if l[1].Kline.Low > h[0].Kline.High || l[1].Kline.Low > h[1].Kline.High {
		return false
	}

	/*
		其他条件,例上升幅度需要大于多少
		或者顶部底部间隔天数等
	*/

	//log.Println(l[0])
	//log.Println(h[0])
	//log.Println(l[1])
	//log.Println(h[1])

	return true
}
