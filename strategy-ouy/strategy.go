package main

import (
	"github.com/injoyai/tdx/extend"
)

// SelectStrategy 选股策略，满足以下4个条件：
// 1. 近20个交易日内出现过一次涨停（这里用涨幅≥9.8%近似判断）
// 2. 涨停之后的下一天出现向上跳空高开（当日开盘价 > 涨停日收盘价）
// 3. 跳空之后出现连续阳线（这里假设至少2根连续阳线，含跳空当天）
// 4. 最新一个交易日的成交量明显放大（> 过去5日均量 且 > 昨日成交量）
func SelectStrategy(klines []*extend.Kline) bool {
	if len(klines) < 20 {
		return false
	}

	// 常量配置，可以根据需要微调
	const LimitUpThreshold = 0.098 // 涨停阈值（9.8%）
	const RecentDaysToCheck = 20
	const ConsecutiveBullDays = 2 // 跳空后要求的连续阳线数量（含跳空当天）

	foundPattern := false

	// 从最近的K线往前检查，限定在最近20个交易日内
	n := len(klines)
	startIndex := n - RecentDaysToCheck
	if startIndex < 0 {
		startIndex = 0
	}
	// 确保有前一日数据用于计算涨跌幅
	if startIndex == 0 {
		startIndex = 1
	}

	// 条件1+2+3：寻找「涨停 → 次日跳空 → 跳空后连续阳线」的形态
	for i := startIndex; i < n-1; i++ {
		curr := klines[i]
		prev := klines[i-1]

		// 条件1：涨停（用收盘价相对昨收涨幅近似判断）
		if prev.Close == 0 {
			continue
		}
		change := (float64(curr.Close) - float64(prev.Close)) / float64(prev.Close)

		if change >= LimitUpThreshold {
			// 找到涨停日，检查下一交易日是否跳空高开
			next := klines[i+1]

			// 条件2：跳空（次日开盘价 > 涨停日收盘价）
			if float64(next.Open) > float64(curr.Close) {
				// 条件3：跳空后有连续阳线，这里从跳空当天开始数
				bullOk := true
				for j := 0; j < ConsecutiveBullDays; j++ {
					idx := i + 1 + j
					if idx >= n {
						bullOk = false
						break
					}
					day := klines[idx]
					// 简单定义：收盘价 > 开盘价 为阳线
					if !(day.Close > day.Open) {
						bullOk = false
						break
					}
				}
				if bullOk {
					foundPattern = true
					break
				}
			}
		}
	}

	if !foundPattern {
		return false
	}

	// 条件4：最近成交量放大
	// 要求：最新一日成交量 > 过去5日平均成交量 且 > 昨日成交量

	lastIndex := n - 1
	lastVol := float64(klines[lastIndex].Volume)

	sumVol := 0.0
	count := 0
	for i := 1; i <= 5; i++ {
		idx := lastIndex - i
		if idx >= 0 {
			sumVol += float64(klines[idx].Volume)
			count++
		}
	}

	if count > 0 {
		avgVol := sumVol / float64(count)
		prevVol := float64(klines[lastIndex-1].Volume)

		if lastVol > avgVol && lastVol > prevVol {
			return true
		}
	}

	return false
}
