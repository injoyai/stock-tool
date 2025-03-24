package main

import (
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"log"
	"time"
)

var (
	testCodes = []string{
		"sh601398",
	}
)

func main() {

	c, err := tdx.DialDefault(nil, tdx.WithRedial())
	logs.PanicErr(err)

	s := &Strategy{
		WindowsSize: 8,
		DayStart:    0,
		DayNumber:   50,
	}

	testCodes = tdx.DefaultCodes.GetStocks()

	result := s.Find(c, testCodes)

	logs.Debug(result)

}

type Strategy struct {
	WindowsSize int    //窗口大小
	DayStart    uint16 //股票起始
	DayNumber   uint16 //股票天数
}

func (this *Strategy) Find(c *tdx.Client, codes []string) []string {

	result := []string(nil)

	for _, code := range codes {

		resp, err := c.GetKlineDay(code, this.DayStart, this.DayNumber)
		logs.PanicErr(err)

		ls := Klines{}
		for _, v := range resp.List {
			ls = append(ls, &Kline{
				Time:  v.Time,
				Open:  v.Open,
				High:  v.High,
				Low:   v.Low,
				Close: v.Close,
			})
		}

		h, l := ls.Vertexes(this.WindowsSize)

		if Check(h, l) {
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

func Check(highs, lows []*Vertex) bool {
	if len(highs) < 2 || len(lows) < 2 {
		return false
	}

	//各取2个最新的顶部和底部
	h := highs[len(highs)-2:]
	l := lows[len(lows)-2:]

	log.Println(l[0])
	log.Println(h[0])
	log.Println(l[1])
	log.Println(h[1])

	//判断顶点是否过远
	if time.Now().Sub(h[1].Kline.Time).Hours()/24 > 10 {
		return false
	}

	//判断时间是否交替
	if !(h[1].Kline.Time.After(l[1].Kline.Time) &&
		l[1].Kline.Time.After(h[0].Kline.Time) &&
		h[0].Kline.Time.After(l[0].Kline.Time)) {
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
