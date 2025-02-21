package main

import (
	"github.com/injoyai/goutil/oss/tray"
)

func main() {

	tray.Run(

		tray.WithStartup(),
		tray.WithSeparator(),
		tray.WithExit(),
	)

}
