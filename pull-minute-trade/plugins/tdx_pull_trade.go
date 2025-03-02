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
		Dir:       filepath.Join(dir, "tdx/trade"),
		Codes:     codes,
		chanPull:  make(chan *ModelPull, limit),
		chanSave:  make(chan *ModelSave, limit),
		limitPull: chans.NewLimit(limit),
		limitSave: chans.NewLimit(limit),
		m:         m,
	}
}

type PullTrade struct {
	Dir       string   //数据保存目录
	Codes     []string //用户指定操作的股票
	chanPull  chan *ModelPull
	chanSave  chan *ModelSave
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

				//1. 打开数据库
				b, err := db.Open(filepath.Join(this.Dir, code+".db"))
				if err != nil {
					logs.Err(err)
					return
				}
				defer b.Close()

				//2. 从数据库获取数据,并删除不全的最后一天数据
				for x := 0; x < 3; x++ {
					last, err := b.GetLast()
					if err != nil {
						logs.Err(err)
						continue
					}
					if last.Time != "15:00" {
						//如果最后时间不是15:00,说明数据不全,删除这天的数据
						if _, err := b.Where("Date=?", last.Date).Delete(&model.Trade{}); err != nil {
							logs.Err(err)
						}
						continue
					}

					//3. 从服务器拉取数据
					var insert []*model.Trade
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

					break
				}

			}(code)

		}

		wg.Wait()

	}

	return nil
}

func (this *PullTrade) pull(c *tdx.Client, code, lastDate string) ([]*model.Trade, error) {
	insert := []*model.Trade(nil)
	t := time.Now()
	now := t.Format("20060102")
	if lastDate == "" {
		lastDate = "19901218"
	}
	for ; lastDate < t.Format("20060102"); t = t.Add(-time.Hour * 24) {
		//<-time.After(time.Millisecond * 100)

		//排除非开市日
		if !this.m.Workday.Is(t) {
			continue
		}
		date := t.Format("20060102")
		logs.Debug(date)

		//如果是当天,获取当天数据,会多个成交单数数据
		if date == now {
			resp, err := c.GetMinuteTradeAll(code)
			if err != nil {
				return nil, err
			}
			before := []*model.Trade(nil)
			for _, v := range resp.List {
				before = append(before, &model.Trade{
					Date:   date,
					Time:   v.Time,
					Price:  v.Price.Int64(),
					Volume: v.Volume,
					Order:  v.Number,
					Status: v.Status,
				})
			}
			insert = append(before, insert...)
			continue
		}

		//获取历史数据
		resp, err := c.GetHistoryMinuteTradeAll(date, code)
		if err != nil {
			return nil, err
		}
		if len(resp.List) == 0 {
			continue
		}
		before := []*model.Trade(nil)
		for _, v := range resp.List {
			before = append(before, &model.Trade{
				Date:   date,
				Time:   v.Time,
				Price:  v.Price.Int64(),
				Volume: v.Volume,
				Order:  0,
				Status: v.Status,
			})
		}
		insert = append(before, insert...)

	}
	return insert, nil
}

//func (this *PullTrade) ReadLastFromDB(ctx context.Context, codes []string, limit *chans.Limit, ch chan *ModelPull) {
//
//	logs.Debug("读取股票数量:", len(codes))
//
//	for i := range codes {
//		select {
//		case <-ctx.Done():
//			return
//
//		default:
//			limit.Add()
//			go func(code string) {
//				defer limit.Done()
//				b, err := db.Open(filepath.Join(this.Dir, code+".db"))
//				if err != nil {
//					logs.Err(err)
//					return
//				}
//				defer b.Close()
//
//				for x := 0; x < 3; x++ {
//					last, err := b.GetLast()
//					if err != nil {
//						logs.Err(err)
//						continue
//					}
//					if last.Time != "15:00" {
//						//如果最后时间不是15:00,说明数据不全,删除这天的数据
//						if _, err := b.Where("Date=?", last.Date).Delete(&db.Trade{}); err != nil {
//							logs.Err(err)
//							continue
//						}
//					}
//					ch <- &ModelPull{
//						Code:  code,
//						Trade: last,
//					}
//					break
//				}
//
//			}(codes[i])
//
//		}
//	}
//}
//
//func (this *PullTrade) PullData(ctx context.Context, m *tdx.Manage, chanPull chan *ModelPull, chanSave chan *ModelSave) {
//	now := time.Now().Format("20060102")
//	for {
//		select {
//		case <-ctx.Done():
//			return
//
//		case data := <-chanPull:
//
//			if data.Updated() {
//				continue
//			}
//
//			m.Go(func(c *tdx.Client) {
//
//				insert := []*db.Trade(nil)
//				data.RangeDate(func(date string) {
//
//					//如果是当天
//					if date == now {
//						resp, err := c.GetMinuteTradeAll(data.Code)
//						if err != nil {
//							logs.Err(err)
//							return
//						}
//
//						for _, v := range resp.List {
//							insert = append(insert, &db.Trade{
//								Date:   date,
//								Time:   v.Time,
//								Price:  v.Price.Int64(),
//								Volume: v.Volume,
//								Order:  v.Number,
//								Status: v.Status,
//							})
//						}
//
//						return
//					}
//
//					//获取历史数据
//					resp, err := c.GetHistoryMinuteTradeAll(date, data.Code)
//					if err != nil {
//						logs.Err(err)
//						return
//					}
//					for _, v := range resp.List {
//						insert = append(insert, &db.Trade{
//							Date:   date,
//							Time:   v.Time,
//							Price:  v.Price.Int64(),
//							Volume: v.Volume,
//							Order:  0,
//							Status: v.Status,
//						})
//					}
//
//				})
//				chanSave <- &ModelSave{
//					Code:   data.Code,
//					Insert: insert,
//				}
//
//			})
//		}
//	}
//}
//
//func (this *PullTrade) SaveToDB(ctx context.Context, limit *chans.Limit, ch chan *ModelSave) {
//	for {
//		select {
//		case <-ctx.Done():
//			return
//
//		case data := <-ch:
//			limit.Add()
//			go func() {
//				defer limit.Done()
//
//				b, err := db.Open(filepath.Join(this.Dir, data.Code+".db"))
//				if err != nil {
//					logs.Err(err)
//					return
//				}
//
//				err = b.SessionFunc(func(session *xorm.Session) error {
//					for _, v := range data.Insert {
//						if _, err := session.Insert(v); err != nil {
//							return err
//						}
//					}
//					return nil
//				})
//				logs.PrintErr(err)
//
//			}()
//
//		}
//	}
//}

type ModelPull struct {
	Code string
	*model.Trade
}

// Updated 返回是否已经更新
func (this *ModelPull) Updated() bool {
	t := time.Now()
	data := t.Format("20060102")
	return this.Date == data && this.Time == t.Format("15:04")
}

func (this *ModelPull) RangeDate(f func(date string)) {
	t := time.Now()
	for ; this.Date < t.Format("20060102"); t.Add(-time.Hour * 24) {
		f(t.Format("20060102"))
	}
}

type ModelSave struct {
	Code   string
	Insert []*model.Trade
}
