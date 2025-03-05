package plugins

import (
	"context"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"sync"
	"time"
	"xorm.io/xorm"
)

var (
	// ExchangeEstablish 交易所成立时间
	ExchangeEstablish = time.Date(1990, 12, 19, 0, 0, 0, 0, time.Local)
)

type PullKline struct {
	Dir   string
	Codes []string
	m     *tdx.Manage
}

func (this *PullKline) Name() string {
	return "更新k线数据"
}

func (this *PullKline) Run(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	for _, v := range this.Codes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(code string) {
			defer wg.Done()

			//1. 打开数据库
			b, err := db.Open(filepath.Join(this.Dir, code+".db"))
			if err != nil {
				logs.Err(err)
				return
			}
			defer b.Close()
			b.Sync2(new(model.DayKline))

			//2. 获取最后一条数据
			last, err := b.GetLastKline()
			if err != nil {
				logs.Err(err)
				return
			}

			//3. 从服务器获取数据
			insert := model.Klines{}
			err = this.m.Do(func(c *tdx.Client) error {
				insert, err = this.pull(c, code, last.Date)
				return err
			})
			if err != nil {
				logs.Err(err)
				return
			}

			//4. 插入数据库
			err = b.SessionFunc(func(session *xorm.Session) error {
				for _, v := range insert {
					if _, err := session.Insert(v); err != nil {
						return err
					}
				}
				return nil
			})
			logs.PrintErr(err)

		}(v)
	}
	return nil
}

func (this *PullKline) pull(c *tdx.Client, code string, lastDate int64) (model.Klines, error) {

	if lastDate == 0 {
		lastDate = ExchangeEstablish.Unix()
	}

	resp, err := c.GetKlineDayUntil(code, func(k *protocol.Kline) bool {
		return k.Time.Unix() <= lastDate
	})
	if err != nil {
		return nil, err
	}

	ks := model.Klines{}
	for _, v := range resp.List {
		ks = append(ks, &model.Kline{
			Date:   v.Time.Unix(),
			Open:   v.Open,
			High:   v.High,
			Low:    v.Low,
			Close:  v.Close,
			Volume: v.Volume,
			Amount: v.Amount,
		})
	}

	return ks, nil
}
