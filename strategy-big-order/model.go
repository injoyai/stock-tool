package main

import (
	"github.com/injoyai/tdx/protocol"
	"time"
)

type Trade struct {
	Time time.Time
	protocol.Price
	Order int
}
