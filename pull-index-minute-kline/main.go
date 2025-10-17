package main

import (
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/tdx"
	"path/filepath"
	"time"
)

var (
	Dir   = "./data/database"
	Codes = []string{
		"sh000001",
		"sz399001",
		"sz399006",
	}
)

func main() {

	for _, code := range Codes {

	}

}

func update(c *tdx.Client, code string) error {
	dir := filepath.Join(Dir, conv.String(time.Now().Year()))
	filename := filepath.Join(dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Desc("")

	return nil
}
