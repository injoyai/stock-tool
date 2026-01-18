package main

import (
	"testing"
	"time"
)

func Test_timeToBytes(t *testing.T) {
	t.Log(timeToBytes(time.Date(2023, 12, 10, 0, 0, 0, 0, time.Local)))
	t.Log(timeToBytes(time.Date(2025, 12, 10, 0, 0, 0, 0, time.Local)))
}
