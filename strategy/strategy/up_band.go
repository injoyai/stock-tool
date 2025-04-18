package strategy

import (
	"log"
	"strategy/model"
)

func NewUpBand(windowSize int, debug bool) *UpBand {
	return &UpBand{
		windowSize: windowSize,
		debug:      debug,
	}
}

type UpBand struct {
	windowSize int
	debug      bool
}

func (this *UpBand) Name() string {
	return "上升波段"
}

func (this *UpBand) Check(ks model.Klines) ([]*model.Point, bool) {

	highs, lows := ks.Vertexes(this.windowSize)

	if len(highs) < 2 || len(lows) < 2 {
		return nil, false
	}

	//各取2个最新的顶部和底部
	h := highs[len(highs)-2:]
	l := lows[len(lows)-2:]

	if this.debug {
		log.Println(l[0].Kline)
		log.Println(h[0].Kline)
		log.Println(l[1].Kline)
		log.Println(h[1].Kline)
	}

	//判断顶点是否过远
	//if int(time.Now().U.Sub(time.Unix(h[1].Kline.Date, 0)).Hours()/24) > this.windowSize*2 {
	//	logs.Err("顶点过远")
	//	return nil,false
	//}

	//判断间隔是否过近
	if h[1].Index-l[1].Index < this.windowSize || l[1].Index-h[0].Index < this.windowSize || h[0].Index-l[0].Index < this.windowSize {
		//logs.Err("顶点过近")
		return nil, false
	}

	//判断时间是否交替
	if !(h[1].Kline.Date > l[1].Kline.Date && h[0].Kline.Date > l[0].Kline.Date && h[0].Kline.Date < l[1].Kline.Date) &&
		!(h[1].Kline.Date < l[1].Kline.Date && h[0].Kline.Date < l[0].Kline.Date && h[1].Kline.Date > l[0].Kline.Date) {
		//logs.Err("顶底不交替")
		return nil, false
	}

	//判断顶部和底部逐步上升
	if h[1].Kline.High <= h[0].Kline.High || l[1].Kline.Low <= l[0].Kline.Low {
		//logs.Err("顶部非逐步上升")
		return nil, false
	}

	//底部不能大于顶部
	if l[1].Kline.Low > h[0].Kline.High || l[1].Kline.Low > h[1].Kline.High {
		//logs.Err("底部大于顶部")
		return nil, false
	}

	/*
		其他条件,例上升幅度需要大于多少
		或者顶部底部间隔天数等
	*/

	//log.Println(l[0])
	//log.Println(h[0])
	//log.Println(l[1])
	//log.Println(h[1])

	res := []*model.Point(nil)
	for _, v := range h {
		res = append(res, &model.Point{
			Index: v.Index,
			Type:  "high",
		})
	}
	for _, v := range l {
		res = append(res, &model.Point{
			Index: v.Index,
			Type:  "low",
		})
	}

	return res, true
}
