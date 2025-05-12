package strategy

import "strategy/model"

type LowPrice struct {
	Window int
}

func (this *LowPrice) Name() string { return "低价" }

func (this *LowPrice) Check(ks model.Klines) ([]*model.Point, bool) { return nil, false }
