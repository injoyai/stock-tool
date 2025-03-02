package plugins

import (
	"context"
	"fmt"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"sync"
	"time"
)

type ExportMinuteKline struct {
	m *tdx.Manage
}

func (this *ExportMinuteKline) Name() string {
	return "导出分时k线数据"
}

func (this *ExportMinuteKline) Run(ctx context.Context) error {
	date := time.Now().Format("20060102")

	codes, err := this.m.Codes.Code(true)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}

	for i := range codes {
		code := codes[i].Code

		wg.Add(1)
		go func() {
			defer wg.Done()

			b, err := db.Open(fmt.Sprintf("./data/database/tdx/trade/%s.db", code))
			if err != nil {
				logs.Err(err)
				return
			}

			data := model.Trades{}
			err = b.Where("Date=?", date).Asc("Time").Find(&data)
			if err != nil {
				logs.Err(err)
				return
			}
		}()

	}

	wg.Wait()

	return nil
}
