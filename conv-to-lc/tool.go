package main

import (
	"math"
	"time"

	"github.com/injoyai/conv"
)

func bytesToFloat(bs [4]byte) float32 {
	return math.Float32frombits(conv.Uint32([]byte{bs[3], bs[2], bs[1], bs[0]}))
}

func floatToBytes(f float32) []byte {
	bs := conv.Bytes(math.Float32bits(f))
	return []byte{bs[3], bs[2], bs[1], bs[0]}
}

func intToBytes(n int32) []byte {
	bs := conv.Bytes(n)
	return []byte{bs[3], bs[2], bs[1], bs[0]}
}

func bytesToTime(bs [4]byte) time.Time {
	n := conv.Int16([]byte{bs[1], bs[0]})

	//2000-2004
	if n >= -8293 && n < 0 {
		year := int(n/2048) + 2004
		month := -(n % 2048) / 100
		day := -(n % 2048) % 100

		num2 := conv.Int16([]byte{bs[3], bs[2]})
		minute := num2 % 60
		hour := num2 / 60

		return time.Date(year, time.Month(month), int(day), int(hour), int(minute), 0, 0, time.Local)
	}

	//2004-2031
	num := conv.Uint16([]byte{bs[1], bs[0]})

	year := int(num/2048) + 2004
	month := (num % 2048) / 100
	month = conv.Select(month < 0, -month, month)
	day := (num % 2048) % 100
	day = conv.Select(day < 0, -day, day)

	num2 := conv.Int16([]byte{bs[3], bs[2]})
	minute := num2 % 60
	hour := num2 / 60

	return time.Date(year, time.Month(month), int(day), int(hour), int(minute), 0, 0, time.Local)
}

func timeToBytes(t time.Time) []byte {

	year, month, day := t.Date()
	n := (year-2004)*2048 +
		int(month)*100 +
		day

	if year < 2004 {
		n = (year-2004)*2048 +
			int(month)*100 +
			day
	}

	dateBs := conv.Bytes(int16(n))

	m := t.Hour()*60 + t.Minute()
	minuteBs := conv.Bytes(int16(m))

	return []byte{dateBs[1], dateBs[0], minuteBs[1], minuteBs[0]}
}
