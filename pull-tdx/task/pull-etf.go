package task

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/injoyai/bar"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

var _ Tasker = new(PullETF)

func NewPullETF(dir string, goroutines int) *PullETF {
	return &PullETF{
		Dir:        dir,
		Goroutines: goroutines,
	}
}

type PullETF struct {
	Dir        string
	Goroutines int
}

func (this *PullETF) Name() string {
	return "拉取ETF"
}

func (this *PullETF) Run(ctx context.Context, m *tdx.Manage) error {
	codes := m.Codes.GetETFCodes()
	b := bar.NewCoroutine(len(codes), this.Goroutines, bar.WithPrefix("xx000000"))
	defer b.Close()

	for i := range codes {
		code := codes[i]
		b.GoRetry(func() error {
			var resp *protocol.KlineResp
			var err error
			err = m.Do(func(c *tdx.Client) error {
				resp, err = c.GetKlineDayAll(code)
				return err
			})
			if err != nil {
				return err
			}
			filename := filepath.Join(this.Dir, fmt.Sprintf("%s.csv", code))
			return klineToCsv(code, resp.List, filename, m.Codes.GetName)
		}, tdx.DefaultRetry)
	}

	b.Wait()

	return nil
}
