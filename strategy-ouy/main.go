package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/extend"
)

// KlineDTO 前端使用的K线数据结构（简化版）
type KlineDTO struct {
	Date   int64   `json:"date"`   // 时间，直接透传底层的时间字段
	Open   float64 `json:"open"`   // 开盘价
	High   float64 `json:"high"`   // 最高价
	Low    float64 `json:"low"`    // 最低价
	Close  float64 `json:"close"`  // 收盘价
	Volume int64   `json:"volume"` // 成交量
}

// StockResult 单只股票的选股结果 + 用于画图的K线数据
type StockResult struct {
	Code   string     `json:"code"`   // 股票代码
	Klines []KlineDTO `json:"klines"` // 最近一段时间的K线数据
}

func main() {

	m, err := tdx.NewManage(
		tdx.WithCodes(nil),
		tdx.WithDialCodes(func(c *tdx.Client) (tdx.ICodes, error) {
			return extend.DialCodesHTTP("http://192.168.192.3:20000")
		}),
		tdx.WithDialGbbq(func(c *tdx.Client) (tdx.IGbbq, error) {
			return extend.DialGbbqHTTP("http://192.168.192.3:20000")
		}),
	)
	logs.PanicErr(err)

	p := extend.NewPullKline(extend.PullKlineConfig{
		Tables: []string{extend.Day},
	})

	//err = p.Run(context.Background(), m)
	//logs.PanicErr(err)
	_ = m

	// Set up HTTP server
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/select", func(w http.ResponseWriter, r *http.Request) {
		handleSelect(w, r, p)
	})

	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	logs.PanicErr(http.ListenAndServe(port, nil))
}

func handleSelect(w http.ResponseWriter, r *http.Request, p *extend.PullKline) {
	// Scan all kline files in the database directory
	files, err := filepath.Glob("./data/database/kline/*.db")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []StockResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency to avoid too many open files or high CPU
	sem := make(chan struct{}, 10)

	// 每只股票返回的最大K线数量，避免一次性返回太多数据
	const maxBarsPerStock = 120

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}

		go func(f string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Extract code from filename (e.g., "sh600000.db" -> "sh600000")
			base := filepath.Base(f)
			code := strings.TrimSuffix(base, ".db")

			// Retrieve klines for the code
			klines, err := p.DayKlines(code)
			if err != nil {
				// logs.Error(err) // reduce log spam
				return
			}

			// Apply the selection strategy
			if SelectStrategy(klines) {
				// 只截取最近一段K线用于前端画图
				start := 0
				if len(klines) > maxBarsPerStock {
					start = len(klines) - maxBarsPerStock
				}
				subset := klines[start:]

				klineDTOs := make([]KlineDTO, 0, len(subset))
				for _, k := range subset {
					// 过滤无效数据：价格为0的情况
					if k.Open <= 0 || k.Close <= 0 || k.High <= 0 || k.Low <= 0 {
						continue
					}

					klineDTOs = append(klineDTOs, KlineDTO{
						Date:   k.Date,
						Open:   float64(k.Open),
						High:   float64(k.High),
						Low:    float64(k.Low),
						Close:  float64(k.Close),
						Volume: k.Volume,
					})
				}

				// 如果过滤后数据太少，可能也不需要展示
				if len(klineDTOs) == 0 {
					return
				}

				mu.Lock()
				result = append(result, StockResult{
					Code:   code,
					Klines: klineDTOs,
				})
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
