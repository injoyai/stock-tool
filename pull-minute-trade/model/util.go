package model

import "time"

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
