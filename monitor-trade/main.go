package main

import (
	"github.com/injoyai/base/g"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"sync"
	"time"
)

var (
	date = time.Now()
)

func main() {

	m, err := tdx.NewManage(&tdx.ManageConfig{
		Number: 1,
	})
	logs.PanicErr(err)

	codes := m.Codes.GetStocks()
	mVolume := make(map[string]Volumes)

	count := uint16(60)
	retry := 3

	{ //计算前n天的平均值
		now := time.Now()
		n := 5
		wg := sync.WaitGroup{}
		for i := range codes {
			code := codes[i]
			wg.Add(1)
			m.Go(func(c *tdx.Client) {
				defer wg.Done()
				for x := 1; x <= n; {
					t := now.AddDate(0, 0, -x)
					if m.Workday.Is(t) {
						x++
						err = g.Retry(func() error {
							resp, err := c.GetHistoryMinuteTrade("", code, 0, count)
							if err == nil {
								mVolume[code] = ToVolumes(resp.List)
							}
							return err
						}, retry)
						logs.PrintErr(err)
					}
				}
			})
		}
		wg.Wait()
	}

	for i := range codes {
		code := codes[i]
		m.Go(func(c *tdx.Client) {
			err = g.Retry(func() error {
				resp, err := c.GetMinuteTrade(code, 0, 60)
			})

		})

	}

}

/*



 */

func ToVolumes(ls []*protocol.HistoryMinuteTrade) Volumes {
	vs := Volumes{}
	for _, v := range ls {
		t, err := time.Parse(time.DateTime, v.Time)
		logs.Err(err)
		vs = append(vs, Volume{
			Volume: v.Volume,
			Time:   t,
			Status: v.Status,
		})
	}
	return vs
}

func ToVolumes2(ls []*protocol.MinuteTrade) Volumes {
	vs := Volumes{}
	for _, v := range ls {
		t, err := time.Parse(time.DateTime, v.Time)
		logs.Err(err)
		vs = append(vs, Volume{
			Volume: v.Volume,
			Time:   t,
			Status: v.Status,
		})
	}
	return vs
}

type Volume struct {
	Volume int
	Time   time.Time
	Status int
}

type Volumes []Volume

func (this Volumes) Before(t time.Time) Volumes {
	res := Volumes{}
	for _, v := range this {
		if v.Time.Before(t) {
			res = append(res, v)
		}
	}
	return res
}

func (this Volumes) Avg() float64 {
	return float64(this.Sum()) / float64(len(this))
}

func (this Volumes) Sum() int {
	total := 0
	for _, v := range this {
		total += v.Volume
	}
	return total
}
