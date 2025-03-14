package task

func NewPullIndex(codes []string) *PullIndex {
	if len(codes) == 0 {
		codes = []string{
			"sh000001",
			"sz399001",
		}
	}
	return &PullIndex{}
}

type PullIndex struct {
	codes []string
}
