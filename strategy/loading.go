package main

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/tdx/extend"
	"path/filepath"
	"strategy/model"
	"time"
)

func NewLoading(dir string) *Loading {
	return &Loading{dir: dir}
}

type Loading struct {
	dir string
}

func (this *Loading) Get(table, code string, start, end time.Time) (model.Klines, error) {
	filename := filepath.Join(this.dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return nil, err
	}
	data := []*extend.Kline(nil)
	err = db.Table(table).Where("Date<=? and Date>=", end.Unix(), start.Unix()).Desc("Date").Find(&data)
	result := model.Klines{}
	for i := len(data) - 1; i >= 0; i-- {
		result = append(result, &model.Kline{
			Index: len(data) - i - 1,
			Kline: data[i],
		})
	}
	return result, nil
}

func (this *Loading) GetBefore(table, code string, t time.Time, number int) (model.Klines, error) {
	filename := filepath.Join(this.dir, code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return nil, err
	}
	data := []*extend.Kline(nil)
	err = db.Table(table).Where("Date<=?", t.Unix()).Limit(number).Desc("Date").Find(&data)
	if err != nil {
		return nil, err
	}
	result := model.Klines{}
	for i := len(data) - 1; i >= 0; i-- {
		result = append(result, &model.Kline{
			Index: len(data) - i - 1,
			Kline: data[i],
		})
	}
	return result, nil
}
