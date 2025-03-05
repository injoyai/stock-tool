package model

import (
	"testing"
	"time"
)

func TestToTime(t *testing.T) {
	t.Log(ToTime(4009, 570))
}

func TestFromTime(t *testing.T) {
	date, minute := FromTime(time.Date(2025, 3, 5, 15, 13, 0, 0, time.Local))
	t.Log(date, minute)
	t.Log(ToTime(date, minute))
}
