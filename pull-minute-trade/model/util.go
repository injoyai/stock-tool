package model

import "time"

// 转时间,最大支持170年,即1990+170=2160
func toTime(date, minute uint16) time.Time {
	year := int(date / 12)
	month := time.Month(date%12 + 1)
	day := int(date & 31)
	return time.Date(year, month, day, int(minute/60), int(minute%60), 0, 0, time.Local)
}

func fromTime(t time.Time) (date uint16, minute uint16) {
	return uint16(t.Year()-1990)*12 + uint16(t.Month()) - 1, 0
}
