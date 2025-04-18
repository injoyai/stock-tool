package main

import (
	"strategy/model"
	"strategy/strategy"
)

func Strategy(ks model.Klines, ss ...strategy.Strategy) (ps []*model.Point, ok bool) {
	for _, s := range ss {
		ps, ok = s.Check(ks)
		if !ok {
			return
		}
	}
	return
}
