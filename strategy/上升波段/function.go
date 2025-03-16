package main

import (
	"fmt"
	"github.com/injoyai/tdx/protocol"
	"time"
)

type Kline struct {
	Time  time.Time      //时间
	Open  protocol.Price //开盘价
	High  protocol.Price //最高价
	Low   protocol.Price //最低价
	Close protocol.Price //收盘价
}

type Point struct {
	Index int    // 数据点索引
	Kline *Kline // 数据点数值
}

func (this Point) String() string {
	return fmt.Sprintf("%v %s %v", this.Index, this.Kline.Time.Format(time.DateOnly), this.Kline.High)
}

type Klines []*Kline

func (this Klines) FindPoint(windowSize int) (highs, lows []Point) {
	if len(this) == 0 {
		return
	}

	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	for i := range this {
		// 动态计算有效窗口边界
		left := max(0, i-windowSize)
		right := min(len(this)-1, i+windowSize)

		// 边缘检测标志
		isLeftEdge := i-windowSize < 0
		isRightEdge := i+windowSize > len(this)-1

		// 极值标记初始化
		isHigh := true
		isLow := true

		// 遍历有效窗口范围
		for j := left; j <= right; j++ {
			if j == i {
				continue // 跳过自身比较
			}

			// 高点检测：存在更高值则取消资格
			if this[j].High >= this[i].High {
				isHigh = false
			}

			// 低点检测：存在更低值则取消资格
			if this[j].Low <= this[i].Low {
				isLow = false
			}
		}

		// 特殊处理数据边界情况
		switch {
		case isLeftEdge && !isRightEdge: // 左边界
			// 只需要比右侧窗口内的高点更高
			if isHigh && this[i].High == maxInWindow(this, i, right) {
				highs = append(highs, Point{i, this[i]})
			}
		case isRightEdge && !isLeftEdge: // 右边界
			// 只需要比左侧窗口内的低点更低
			if isLow && this[i].Low == minInWindow(this, left, i) {
				lows = append(lows, Point{i, this[i]})
			}
		default: // 正常区间
			if isHigh {
				highs = append(highs, Point{i, this[i]})
			}
			if isLow {
				lows = append(lows, Point{i, this[i]})
			}
		}
	}
	return
}

// 辅助函数：计算窗口内最大值
func maxInWindow(prices []*Kline, start, end int) protocol.Price {
	maxVal := prices[start].High
	for i := start + 1; i <= end; i++ {
		if prices[i].High > maxVal {
			maxVal = prices[i].High
		}
	}
	return maxVal
}

// 辅助函数：计算窗口内最小值
func minInWindow(prices []*Kline, start, end int) protocol.Price {
	minVal := prices[start].Low
	for i := start + 1; i <= end; i++ {
		if prices[i].Low < minVal {
			minVal = prices[i].Low
		}
	}
	return minVal
}
