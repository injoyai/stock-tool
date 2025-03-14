package task

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"time"
	"xorm.io/xorm"
)

var (
	// ExchangeEstablish 交易所成立时间
	ExchangeEstablish = time.Date(1990, 12, 19, 0, 0, 0, 0, time.Local)
)

func NewPullKline(codes []string, dir string, limit int) *PullKline {
	return &PullKline{
		Dir:   dir,
		Codes: codes,
		limit: limit,
	}
}

type PullKline struct {
	Dir   string
	Codes []string //指定的代码
	limit int      //并发数量
}

func (this *PullKline) Name() string {
	return "更新k线数据"
}

func (this *PullKline) Run(ctx context.Context, m *tdx.Manage) error {
	limit := chans.NewWaitLimit(uint(this.limit))
	for _, v := range this.Codes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		limit.Add()
		go func(code string) {
			defer limit.Done()

			tables := []*model.KlineTable{
				model.NewKlineTable("MinuteKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineMinuteUntil }),
				model.NewKlineTable("Minute5Kline", func(c *tdx.Client) model.KlineHandler { return c.GetKline5MinuteUntil }),
				model.NewKlineTable("DayKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineDayUntil }),
				model.NewKlineTable("WeekKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineWeekUntil }),
				model.NewKlineTable("MonthKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineMonthUntil }),
				model.NewKlineTable("QuarterKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineQuarterUntil }),
				model.NewKlineTable("YearKline", func(c *tdx.Client) model.KlineHandler { return c.GetKlineYearUntil }),
			}

			//1. 打开数据库
			b, err := db.Open(filepath.Join(this.Dir, code+".db"))
			if err != nil {
				logs.Err(err)
				return
			}
			defer b.Close()
			for _, table := range tables {
				select {
				case <-ctx.Done():
					return
				default:
				}

				b.Sync2(table)

				//2. 获取最后一条数据
				last, err := b.GetLastKline(table)
				if err != nil {
					logs.Err(err)
					return
				}

				//3. 从服务器获取数据
				insert := model.Klines{}
				err = m.Do(func(c *tdx.Client) error {
					insert, err = this.pull(code, last.Date, table.Handler(c))
					return err
				})
				if err != nil {
					logs.Err(err)
					return
				}

				//4. 插入数据库
				err = b.SessionFunc(func(session *xorm.Session) error {
					for i, v := range insert {
						if i == 0 {
							if _, err := session.Table(table).Where("Date >= ?", v.Date).Delete(); err != nil {
								return err
							}
						}
						if _, err := session.Table(table).Insert(v); err != nil {
							return err
						}
					}
					return nil
				})
				logs.PrintErr(err)

			}

		}(v)
	}
	limit.Wait()
	return nil
}

func (this *PullKline) pull(code string, lastDate int64, f func(code string, f func(k *protocol.Kline) bool) (*protocol.KlineResp, error)) (model.Klines, error) {

	if lastDate == 0 {
		lastDate = ExchangeEstablish.Unix()
	}

	resp, err := f(code, func(k *protocol.Kline) bool {
		return k.Time.Unix() <= lastDate
	})
	if err != nil {
		return nil, err
	}

	ks := model.Klines{}
	for _, v := range resp.List {
		ks = append(ks, &model.Kline{
			Code:   code,
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
