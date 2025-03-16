package task

func NewPullIndex(codes []string) *PullIndex {
	if len(codes) == 0 {
		codes = []string{
			"sh000001", //上证指数
			"sz399001", //深证成指
			"sh000016", //上证50
			"sh000010", //上证180
			"sh000300", //上证300 sz399300
			"sh000905", //中证500
			"sh000852", //中证1000
			"sz399006", //创业板指
		}
	}
	return &PullIndex{}
}

type PullIndex struct {
	codes []string
}
