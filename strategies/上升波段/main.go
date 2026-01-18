package main

import (
	"bytes"
	_ "embed"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/frame/in/v3"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/shell"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
)

var (
	testCodes = []string{
		//"sz000400",
	}
	debug = len(testCodes) > 0
)

//go:embed kline.html
var index []byte

func main() {

	//RunHTTP(8080)

	s := &Strategy{
		WindowsSize: 8,
		DayEnd:      time.Now(), //time.Date(2025, 12, 10, 0, 0, 0, 0, time.Local),
		DayNumber:   100,
		Dir:         filepath.Join(tdx.DefaultDatabaseDir, "daykline"),
	}

	c, err := tdx.DialDefault()
	logs.PanicErr(err)

	codes, err := tdx.NewCodes(c, filepath.Join(tdx.DefaultDatabaseDir, "codes.db"))
	logs.PanicErr(err)

	if len(testCodes) == 0 {
		oss.RangeFileInfo(s.Dir, func(info *oss.FileInfo) (bool, error) {
			code := strings.SplitN(info.Name(), ".", 2)[0]
			switch {
			case strings.HasPrefix(code, "sh") || strings.HasPrefix(code, "sz0"):
				testCodes = append(testCodes, code)
			}
			return true, nil
		})
	}

	result := s.Find(testCodes)

	for _, v := range result {
		v.Name = codes.GetName(v.Name) + "(" + v.Name + ")"
	}

	RunHTTP(8090, index, result)

}

type Strategy struct {
	WindowsSize int       //窗口大小
	DayStart    uint16    //股票起始
	DayNumber   int       //股票天数
	DayEnd      time.Time //
	Dir         string
}

func (this *Strategy) GetKlines(code string, limit int) (Klines, error) {
	data := []*extend.Kline(nil)
	db, err := sqlite.NewXorm(filepath.Join(this.Dir, code+".db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.Table(extend.NewKlineTable("DayKline", nil)).Desc("Date").Where("Date<?", this.DayEnd.Unix()).Limit(limit).Find(&data)
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

func (this *Strategy) Find(codes []string) []*Result {

	result := []*Result(nil)

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
			result = append(result, &Result{
				Name: code,
				Data: &Data{
					Data: func() [][5]string {
						res := [][5]string(nil)
						for _, v := range ls {
							res = append(res, [5]string{
								v.Time.Format("2006-01-02"),
								conv.String(v.Open.Float64()),
								conv.String(v.Close.Float64()),
								conv.String(v.Low.Float64()),
								conv.String(v.High.Float64()),
							})
						}
						return res
					}(),
					Points: func() []*Point {
						res := []*Point(nil)
						for _, v := range h {
							res = append(res, &Point{
								Index: v.Index,
								Type:  "high",
							})
						}
						for _, v := range l {
							res = append(res, &Point{
								Index: v.Index,
								Type:  "low",
							})
						}
						return res
					}(),
				},
			})
			logs.Debug(code)
		}

	}

	return result
}

type Result struct {
	Name string `json:"name"`
	Data *Data  `json:"data"`
}

type Data struct {
	Data   [][5]string `json:"data"`
	Points []*Point    `json:"markPoints"`
}

type Point struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
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
		log.Println("h[0]", h[0].Kline)
		log.Println("l[0]", l[0].Kline)
		log.Println("h[1]", h[1].Kline)
		log.Println("l[1]", l[1].Kline)
	}

	//判断顶点是否过远
	//if int(h[1].Time.Sub(h[0].Kline.Time).Hours()/24) > windowSize*2 {
	//	return false
	//}

	//判断间隔是否过近
	if l[1].Index-h[1].Index < windowSize || h[1].Index-l[0].Index < windowSize || l[0].Index-h[0].Index < windowSize {
		return false
	}

	////判断时间是否交替
	//if !(h[1].Kline.Time.After(l[1].Kline.Time) &&
	//	l[1].Kline.Time.After(h[0].Kline.Time) &&
	//	h[0].Kline.Time.After(l[0].Kline.Time)) {
	//	return false
	//}

	//判断时间是否交替
	if !(l[1].Kline.Time.After(h[1].Kline.Time) &&
		h[1].Kline.Time.After(l[0].Kline.Time) &&
		l[0].Kline.Time.After(h[0].Kline.Time)) {
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

func RunHTTP(port int, index []byte, data any) error {
	s := mux.New()
	s.SetPort(port)
	s.GET("/", func(r *mux.Request) {
		bs := bytes.ReplaceAll(index, []byte("{port}"), []byte(conv.String(port)))
		in.Html200(bs)
	})
	s.GET("/data.json", func(r *mux.Request) {
		in.Json200(data)
	})
	go func() {
		<-time.After(time.Millisecond * 100)
		shell.OpenBrowser("http://127.0.0.1:" + strconv.Itoa(port))
	}()
	return s.Run()
}
