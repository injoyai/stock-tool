package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/injoyai/tdx/protocol"
)

type Kline struct {
	Time  time.Time      //时间
	Open  protocol.Price //开盘价
	High  protocol.Price //最高价
	Low   protocol.Price //最低价
	Close protocol.Price //收盘价
}

type Klines []*Kline

func (this Klines) String() string {
	ls := []string(nil)
	for _, v := range this {
		ls = append(ls, fmt.Sprintf("max:%.2f, min:%.2f", v.High.Float64(), v.Low.Float64()))
	}
	return strings.Join(ls, "\n")
}

// IsVertex 判断当前值是否是顶点(最大值/最小值)
func (this Klines) IsVertex(k *Kline) (bool, bool) {
	isMax := true
	isMin := true
	//判断当前值是否是最大值/最小值
	for _, v := range this {
		if k == v {
			continue
		}
		if k.High <= v.High {
			isMax = false
		}
		if k.Low >= v.Low {
			isMin = false
		}
	}
	return isMax, isMin
}

func (this Klines) Vertexes(windowSize int, filterEdge ...bool) (maxs []*Vertex, mins []*Vertex) {

	filter := false
	if len(filterEdge) > 0 {
		filter = filterEdge[0]
	}

	for i := 0; i < len(this); i++ {

		var cache Klines

		switch {
		case i-windowSize < 0 && len(this)-i < windowSize:
			//左边右边都不满足窗口大小
			if filter {
				continue
			}
			cache = this

		case i-windowSize < 0:
			//左侧不满足窗口大小
			if filter {
				continue
			}
			cache = this[:i+windowSize]

		case len(this)-1-i < windowSize:
			//右侧不满足窗口大小
			if filter {
				continue
			}
			if len(this[i:]) < windowSize/2 {
				continue
			}
			cache = this[i-windowSize:]

		default:
			//满足左边右边都有窗口大小
			cache = this[i-windowSize : i+windowSize+1]

		}

		//logs.Debug()
		//fmt.Println(cache)

		//判断当前值是否是最大值/最小值
		isMax, isMin := cache.IsVertex(this[i])
		if isMax {
			maxs = append(maxs, &Vertex{Index: i, Kline: this[i]})
		}
		if isMin {
			mins = append(mins, &Vertex{Index: i, Kline: this[i]})
		}

	}

	return
}

type Vertex struct {
	Index int
	*Kline
}
