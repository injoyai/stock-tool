package main

import (
	"github.com/injoyai/base/types"
	"github.com/injoyai/tdx"
	"sort"
	"time"
)

// ToTime 转时间,最大支持170年,即1990+170=2160
func ToTime(date, minute uint16) time.Time {
	yearMonth := date >> 5
	year := int(yearMonth/12) + 1990
	month := time.Month(yearMonth%12 + 1)
	day := int(date & 31)
	return time.Date(year, month, day, int(minute/60), int(minute%60), 0, 0, time.Local)
}

// FromTime x
func FromTime(t time.Time) (date uint16, minute uint16) {
	return (uint16(t.Year()-1990)*12+uint16(t.Month()-1))<<5 + uint16(t.Day()), uint16(t.Hour()*60 + t.Minute())
}

func getPublic(m *tdx.Manage, code string) (year int, month time.Month, err error) {
	year = 1990
	month = 12
	err = m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineMonthAll(code)
		if err != nil {
			return err
		}
		if len(resp.List) > 0 {
			year = resp.List[0].Time.Year()
			month = resp.List[0].Time.Month()
			return nil
		}
		return nil
	})
	return
}

func GetCodes(m *tdx.Manage, codes []string) []string {
	if len(codes) == 0 {
		return m.Codes.GetStocks()
	}
	return codes
}

type Map[K types.Comparable, V any] map[K]V

func (this Map[K, V]) Sort() []V {
	items := make([]item[K, V], 0, len(this))
	for k, v := range this {
		items = append(items, item[K, V]{
			K: k,
			V: v,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].K < items[j].K
	})
	ret := make([]V, 0, len(items))
	for _, item := range items {
		ret = append(ret, item.V)
	}
	return ret
}

type item[K comparable, V any] struct {
	K K
	V V
}
