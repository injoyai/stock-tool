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

	c, err := tdx.DialHostsRandom(nil, tdx.WithRedial())
	logs.PanicErr(err)

	s := &Strategy{
		WindowsSize: 8,
		DayStart:    0,
		DayNumber:   50,
	}

	testCodes = codes.GetStocks()

	result := s.Find(c, testCodes)

	logs.Debug(result)

}

type Result struct {
	Points [4]Point
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

		h, l := ls.FindPoint(this.WindowsSize)

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

func Check(highs, lows []Point) bool {
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

//type Klines []*Kline
//
//// FindPoint ，windowSize为0时使用基础模式
//func (this Klines) FindPoint(windowSize int) (highs, lows []Point) {
//	if len(this) == 0 {
//		return
//	}
//
//	for i := range this {
//		// 计算有效窗口范围
//		start := int(math.Max(0, float64(i-windowSize)))
//		end := int(math.Min(float64(len(this)-1), float64(i+windowSize)))
//
//		// 极值标记
//		isHigh := true
//		isLow := true
//
//		// 窗口内遍历
//		for j := start; j <= end; j++ {
//			if i == j {
//				continue // 跳过自身比较
//			}
//
//			if this[j].High > this[i].High {
//				isHigh = false // 存在更高值则不是高点
//			}
//
//			if this[j].Low < this[i].Low {
//				isLow = false // 存在更低值则不是低点
//			}
//
//			// 提前退出优化：当两个标记都为false时停止检查
//			if !isHigh && !isLow {
//				break
//			}
//		}
//
//		// 记录有效极值
//		if isHigh {
//			highs = append(highs, Point{i, this[i]})
//		}
//		if isLow {
//			lows = append(lows, Point{i, this[i]})
//		}
//	}
//	return
//}
