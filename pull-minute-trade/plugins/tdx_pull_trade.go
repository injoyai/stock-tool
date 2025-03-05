package plugins

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-minute-trade/db"
	"pull-minute-trade/model"
	"sync"
	"time"
	"xorm.io/xorm"
)

func NewPullTrade(m *tdx.Manage, codes []string, dir string, limit int) *PullTrade {
	return &PullTrade{
		Dir:       filepath.Join(dir, "trade"),
		Codes:     codes,
		limitPull: chans.NewLimit(limit),
		limitSave: chans.NewLimit(limit),
		m:         m,
	}
}

type PullTrade struct {
	Dir       string   //数据保存目录
	Codes     []string //用户指定操作的股票
	limitPull *chans.Limit
	limitSave *chans.Limit
	m         *tdx.Manage
}

func (this *PullTrade) Name() string {
	return "更新交易数据"
}

func (this *PullTrade) Running() bool {
	return false
}

func (this *PullTrade) RunInfo() string {
	return ""
}

func (this *PullTrade) Run(ctx context.Context) error {

	var wg sync.WaitGroup

	select {
	case <-ctx.Done():
		return ctx.Err()

	default:

		//1. 获取所有股票代码
		codes := this.Codes
		if len(codes) == 0 {
			codes = this.m.Codes.GetStocks()
		}

		for _, code := range codes {

			select {
			case <-ctx.Done():
				return ctx.Err()

			default:
			}

			wg.Add(1)
			go func(code string) {
				defer wg.Done()
				logs.Debug("开始更新:", code)
				logs.Debug(filepath.Join(this.Dir, code+".db"))

				//1. 打开数据库
				b, err := db.Open(filepath.Join(this.Dir, code+".db"))
				if err != nil {
					logs.Err(err)
					return
				}
				defer b.Close()
				b.Sync2(new(model.Trade))

				//2. 从数据库获取数据,并删除不全的最后一天数据
				for x := 0; x < 3; x++ {
					last, err := b.GetLastTrade()
					if err != nil {
						logs.Err(err)
						continue
					}
					if last.Time != 0 && last.Time != 900 {
						//如果最后时间不是15:00,说明数据不全,删除这天的数据
						if _, err := b.Where("Date=?", last.Date).Delete(&model.Trade{}); err != nil {
							logs.Err(err)
							continue
						}
					}

					if last.Date == 0 {
						last.Date, _ = model.FromTime(ExchangeEstablish)
					}

					//解析日期
					now := time.Now()
					t := model.ToTime(last.Date, 0)

					var insert []*model.Trade
					//遍历时间,并加入数据库
					for start := t.Add(time.Hour * 24); start.Before(now); start = start.Add(time.Hour * 24) {
						//3. 获取数据
						err = this.m.Do(func(c *tdx.Client) error {
							ls, err := this.pullDay(c, code, start, now)
							if err != nil {
								return err
							}
							insert = append(insert, ls...)
							return nil
						})
						if err != nil {
							logs.Err(err)
							return
						}

						//排除数据为0的,可能这天停牌了啥的
						if len(insert) == 0 {
							continue
						}

						//4. 插入数据库
						if len(insert) > 1e6 {
							err = b.SessionFunc(func(session *xorm.Session) error {
								for _, v := range insert {
									if _, err := session.Insert(v); err != nil {
										return err
									}
								}
								return nil
							})
							if err != nil {
								logs.Err(err)
								return
							}
							insert = insert[:0]
						}
					}

					if len(insert) > 0 {
						err = b.SessionFunc(func(session *xorm.Session) error {
							for _, v := range insert {
								if _, err := session.Insert(v); err != nil {
									return err
								}
							}
							return nil
						})
						if err != nil {
							logs.Err(err)
							return
						}
						insert = insert[:0]
					}

					break
				}

			}(code)

		}

		wg.Wait()

	}

	return nil
}

func (this *PullTrade) pullDay(c *tdx.Client, code string, start, now time.Time) ([]*model.Trade, error) {

	//排除休息日
	if !this.m.Workday.Is(start) {
		return nil, nil
	}

	insert := []*model.Trade(nil)

	date, _ := model.FromTime(start)
	nowDate, _ := model.FromTime(now)
	startTime := time.Now()
	defer func() {
		logs.Debug(start.Format("2006-01-02"), "耗时:", time.Since(startTime), "数据数量:", len(insert))
	}()

	switch date {
	case 0:
		//

	case nowDate:
		//如果是当天,获取当天数据,会多个成交单数数据
		resp, err := c.GetMinuteTradeAll(code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			t, err := time.ParseInLocation("15:04", v.Time, time.Local)
			if err != nil {
				return nil, err
			}
			_, minute := model.FromTime(t)
			insert = append(insert, &model.Trade{
				Date:   date,
				Time:   minute,
				Price:  v.Price.Int64(),
				Volume: v.Volume,
				Order:  v.Number,
				Status: v.Status,
			})
		}

	default:
		//获取历史数据
		resp, err := c.GetHistoryMinuteTradeAll(start.Format("20060102"), code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			t, err := time.ParseInLocation("15:04", v.Time, time.Local)
			if err != nil {
				return nil, err
			}
			_, minute := model.FromTime(t)
			insert = append(insert, &model.Trade{
				Date:   date,
				Time:   minute,
				Price:  v.Price.Int64(),
				Volume: v.Volume,
				Order:  0,
				Status: v.Status,
			})
		}

	}

	return insert, nil
}
