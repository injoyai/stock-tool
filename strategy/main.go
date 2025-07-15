package main

import (
	_ "embed"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/frame/in/v3"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/oss/shell"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"strategy/model"
	"strategy/strategy"
	"strconv"
	"strings"
	"time"
)

func main() {

	//Pull([]string{extend.Day})

	endAt := time.Now().AddDate(0, 0, 0)
	offset := 0
	number := 120
	windowSize := 15
	codes := []string{
		//"sz000001",
	}
	if len(codes) == 0 {
		oss.RangeFileInfo(filepath.Join(tdx.DefaultDatabaseDir, "daykline"), func(info *oss.FileInfo) (bool, error) {
			codes = append(codes, strings.Split(info.Name(), ".")[0])
			return true, nil
		})
	}
	tables := []string{"DayKline"}
	strategies := []strategy.Strategy{
		strategy.NewUpBand(windowSize, false),
	}

	/*



	 */

	//更新数据
	//logs.Debug("更新数据")
	//_, err := Pull(tables)
	//logs.PanicErr(err)
	cs, err := tdx.DialCodes(filepath.Join(tdx.DefaultDatabaseDir, "codes.db"))
	logs.PanicErr(err)

	//加载数据
	logs.Debug("加载数据")
	result := []*model.Result(nil)
	l := NewLoading(filepath.Join(tdx.DefaultDatabaseDir, "daykline"))
	for _, table := range tables {
		for _, code := range codes {
			ks, err := l.GetBefore(table, code, endAt, number)
			logs.PanicErr(err)

			if len(ks) <= offset {
				continue
			}

			//执行策略
			if ps, ok := Strategy(ks[:len(ks)-offset], strategies...); ok {
				result = append(result, &model.Result{
					Name: cs.GetName(code) + "(" + code + ")",
					Data: &model.Data{
						Data: func() [][5]string {
							res := [][5]string(nil)
							for _, v := range ks {
								res = append(res, [5]string{
									time.Unix(v.Date, 0).Format(time.DateOnly),
									conv.String(v.Open.Float64()),
									conv.String(v.Close.Float64()),
									conv.String(v.Low.Float64()),
									conv.String(v.High.Float64()),
								})
							}
							return res
						}(),
						Points: ps,
					},
				})
			}
		}
	}

	logs.Debug("打开视图")
	RunHTTP(8080, result)
}

//go:embed index.html
var index []byte

func RunHTTP(port int, data any) error {
	s := mux.New()
	s.SetPort(port)
	s.GET("/", func(r *mux.Request) {
		in.Html200(index)
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

//type Result struct {
//	Name string `json:"name"`
//	Data *Data  `json:"data"`
//}
//
//type Data struct {
//	Data   [][5]string `json:"data"`
//	Points []*Point    `json:"markPoints"`
//}
//
//type Point struct {
//	Index int    `json:"index"`
//	Type  string `json:"type"`
//}
