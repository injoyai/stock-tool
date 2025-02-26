package db

import (
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"time"
)

type Sqlite struct {
	err error
	*xorms.Engine
}

func (this *Sqlite) Save(table string) {

}

func (this *Sqlite) Find(table string) {

}

// GetLast 获取最后一条数据
func (this *Sqlite) GetLast() (*Model, error) {
	if this.err != nil {
		return nil, this.err
	}
	data := new(Model)
	_, err := this.Desc("date", "time").Get(data)
	return data, err
}

// Open 打开数据库
func Open(filename string) *Sqlite {
	db, err := sqlite.NewXorm(filename)
	return &Sqlite{
		err:    err,
		Engine: db,
	}
}

type Model struct {
	Date   string //日期
	Time   string //时间
	Price  int64  //成交价格,单位分
	Volume int    //交易量
	Status uint8  //买或者卖
}

type Message struct {
	Code string
	*Model
}

// Updated 返回是否已经更新
func (this *Message) Updated() bool {
	t := time.Now()
	data := t.Format("20060102")
	return this.Date == data && this.Time == t.Format("15:04")
}

func (this *Message) RangeDate(f func(date string)) {
	t := time.Now()
	for ; this.Date <= t.Format("20060102"); t.Add(-time.Hour * 24) {

	}
}
