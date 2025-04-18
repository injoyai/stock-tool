package model

import (
	"testing"
	"time"
)

/*
4009: 2000-06-09   		570: 09:30
13510: 2025-03-06
7547: 2009-08-27
*/
func TestToTime(t *testing.T) {
	t.Log(ToTime(4009, 570))
	t.Log(ToTime(13510, 570))
	t.Log(ToTime(4016, 899)) //sh600612
}

func TestFromTime(t *testing.T) {
	date, minute := FromTime(time.Date(2010, 2, 12, 15, 13, 0, 0, time.Local))
	t.Log(date, minute)
	t.Log(ToTime(date, minute))
}
