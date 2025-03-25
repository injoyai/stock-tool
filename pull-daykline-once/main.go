package main

import "time"

func main() {
	defer done()()

	bySector(
		[]string{},
		time.Date(2016, 1, 1, 0, 0, 0, 0, time.Local),
		time.Date(2025, 3, 24, 23, 0, 0, 0, time.Local),
	)

	//byStock(
	//	[]string{},
	//	time.Date(2016, 1, 1, 0, 0, 0, 0, time.Local),
	//	time.Date(2025, 3, 23, 23, 0, 0, 0, time.Local),
	//)

}
