package main

import (
	"github.com/injoyai/tdx"
	"time"
)

func main() {

	pull(nil, time.Time{}, time.Now().Add(time.Hour*24), func(c *tdx.Client) Handler { return c.GetKline15MinuteUntil })

}
