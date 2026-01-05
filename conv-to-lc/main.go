package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
)

const (
	Coroutines = 10
)

var (
	start = time.Date(2004, 1, 1, 0, 0, 0, 0, time.Local)
	end   = time.Now()
)

func main() {
	tool()
}

func tool() {

	var err error

	for {
		startStr := g.Input("请输入开始日期(默认" + start.Format("20060102") + "):")
		if len(startStr) == 0 {
			break
		}
		start, err = time.Parse("20060102", startStr)
		if err != nil {
			logs.Err(err)
			continue
		}
		break
	}

	for {
		endStr := g.Input("请输入结束日期(默认" + end.Format("20060102") + "):")
		if len(endStr) == 0 {
			break
		}
		end, err = time.Parse("20060102", endStr)
		if err != nil {
			logs.Err(err)
			continue
		}
		break
	}

	goroutines := g.InputVar("请输入协程数(默认10):").Int(Coroutines)
	after := g.Input("从哪里开始(例sh600000):")

	logs.Info("开始转换5分钟...")
	err = _conv(
		"./5分钟",
		fmt.Sprintf("./lc5(%d-%d)/", start.Year(), end.Year()),
		".lc5",
		start,
		end,
		goroutines,
		after,
	)
	logs.PrintErr(err)

	logs.Info("开始转换1分钟...")
	err = _conv(
		"./1分钟",
		fmt.Sprintf("./lc1(%d-%d)/", start.Year(), end.Year()),
		".lc1",
		start,
		end,
		goroutines,
		after,
	)
	logs.PrintErr(err)

	g.Input("按回车键结束...")

}

func _conv(inputDir, outputDir, suffix string, start, end time.Time, goroutines int, after string) error {
	os.MkdirAll(outputDir, 0666)

	ls, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	b := bar.NewCoroutine(len(ls), goroutines,
		bar.WithPrefix("[xx000000]"),
		bar.WithFlush(),
	)
	defer b.Close()

	oss.RangeFileInfo(
		inputDir,
		func(info *oss.FileInfo) (bool, error) {
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".csv") {
				return true, nil
			}
			b.Go(func() {
				if info.Name() < after {
					return
				}
				name := strings.Split(info.Name(), ".")[0]
				b.SetPrefix("[" + name + "]")
				b.Flush()
				err = convLc(info.FullName(), filepath.Join(outputDir, name+suffix), start, end)
				if err != nil {
					b.Logf("[错误] %s %v", info.Name(), err)
					b.Flush()
				}
			})
			return true, nil
		},
	)

	b.Wait()

	return nil
}

/*
inputFile := "./data/1分钟/sh600000-5min.csv"

	outputFile := "./data/lc1/sh600000.lc5"
*/
func convLc(inputFile, outputFile string, start, end time.Time) error {

	// 打开 CSV
	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	//按年分
	for i, row := range rows {
		// 跳过表头，从第 2 行开始
		if i == 0 {
			continue
		}

		t, err := time.Parse("2006-01-02 15:04", row[0]+" "+row[1])
		if err != nil {
			return err
		}

		if t.Before(start) || t.After(end) {
			continue
		}

		_, err = out.Write(timeToBytes(t)) //4字节,时间
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[2]))) //4字节,开盘
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[3]))) //4字节,最高
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[4]))) //4字节,最低
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[5]))) //4字节,收盘
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[7]))) //4字节,成交额,股
		if err != nil {
			return err
		}

		_, err = out.Write(intToBytes(conv.Int32(row[6]) * 100)) //4字节,成交量,元
		if err != nil {
			return err
		}

		_, err = out.Write([]byte{0, 0, 0, 0}) //4字节,预留
		if err != nil {
			return err
		}

	}

	return nil
}

/*








 */

func convIndex(inputDir, outputDir, suffix string, start, end time.Time) error {
	os.MkdirAll(outputDir, 0666)

	ls, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	b := bar.New(
		bar.WithTotal(int64(len(ls))),
		bar.WithPrefix("[shxxxxxx]"),
		bar.WithFormat(
			bar.WithPlan(),
			bar.WithRateSize(),
			bar.WithSpeed(),
			bar.WithRemain(),
		),
		bar.WithFlush(),
	)

	defer b.Close()
	return oss.RangeFileInfo(
		inputDir,
		func(info *oss.FileInfo) (bool, error) {
			defer func() {
				b.Add(1)
				b.Flush()
			}()
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".csv") {
				return true, nil
			}
			name := strings.Split(info.Name(), ".")[0]
			b.SetPrefix("[" + name + "]")
			b.Flush()
			err = convLcIndex(info.FullName(), filepath.Join(outputDir, name+suffix), start, end)
			if err != nil {
				b.Logf("[错误] %s %v", info.Name(), err)
			}
			return true, err
		},
	)
}

func convLcIndex(inputFile, outputFile string, start, end time.Time) error {

	// 打开 CSV
	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	//按年分
	for i, row := range rows {
		// 跳过表头，从第 2 行开始
		if i == 0 {
			continue
		}

		t, err := time.Parse("2006-01-02 15:04", row[0]+" "+row[1])
		if err != nil {
			return err
		}

		if t.Before(start) || t.After(end) {
			continue
		}

		_, err = out.Write(timeToBytes(t)) //4字节,时间
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[2]))) //4字节,开盘
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[3]))) //4字节,最高
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[4]))) //4字节,最低
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[5]))) //4字节,收盘
		if err != nil {
			return err
		}

		_, err = out.Write(floatToBytes(conv.Float32(row[7]))) //4字节,成交额,股
		if err != nil {
			return err
		}

		_, err = out.Write(intToBytes(conv.Int32(row[7]) * 100)) //4字节,成交量,元
		if err != nil {
			return err
		}

		_, err = out.Write([]byte{0, 0, 0, 0}) //4字节,预留
		if err != nil {
			return err
		}

	}

	return nil
}
