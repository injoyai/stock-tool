package task

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

func NewPullIndex(dir string, codes map[string]string) *PullIndex {
	if len(codes) == 0 {
		codes = map[string]string{
			"sh000001": "上证指数",
			"sz399001": "深证成指",
			"sh000016": "上证50",
			"sh000688": "科创50",
			"sh000010": "上证180",
			"sh000300": "沪深300",
			"sh000905": "中证500",
			"sh000852": "中证1000",
			"sz399006": "创业板指",
			"sh000932": "中证消费指数",
			"sh000827": "中证环保指数",
		}
	}
	return &PullIndex{
		Dir:   dir,
		Codes: codes,
	}
}

type PullIndex struct {
	Dir   string
	Codes map[string]string
}

func (this *PullIndex) Name() string {
	return "拉取指数"
}

func (this *PullIndex) Run(ctx context.Context, m *tdx.Manage) error {
	return m.Do(func(c *tdx.Client) error {
		this.pull(ctx, c.GetIndexDayAll, "日线")
		this.pull(ctx, c.GetIndexWeekAll, "周线")
		this.pull(ctx, c.GetIndexMonthAll, "月线")
		this.pull(ctx, c.GetIndexQuarterAll, "季线")
		this.pull(ctx, c.GetIndexYearAll, "年线")
		return nil
	})
}

func (this *PullIndex) pull(ctx context.Context, f func(code string) (*protocol.KlineResp, error), class string) error {
	logs.Tracef("开始拉取%s...\n", class)
	for code, name := range this.Codes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			filename := filepath.Join(this.Dir, class, fmt.Sprintf("%s(%s).csv", code, name))
			resp, err := f(code)
			if err != nil {
				return err
			}
			err = klineToCsv(code, resp.List, filename, func(string) string { return name })
			if err != nil {
				return err
			}
		}
	}
	return nil
}
