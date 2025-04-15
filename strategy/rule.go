package strategy

type Rule interface {
}

type upBand struct {
}

func (this *upBand) Name() string {
	return "上升波段"
}
