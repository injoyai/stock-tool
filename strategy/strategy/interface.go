package strategy

import (
	"strategy/model"
)

type Strategy interface {
	Check(ks model.Klines) ([]*model.Point, bool)
}
