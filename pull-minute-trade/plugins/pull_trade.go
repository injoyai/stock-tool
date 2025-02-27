package plugins

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-minute-trade/db"
	"sync"
)

func NewPullTrade(m *tdx.Manage, dir string, limit int) *PullTrade {
	return &PullTrade{
		Dir:       dir,
		chanGet:   make(chan *db.Message, limit),
		chanSave:  make(chan *db.Message, limit),
		limitGet:  chans.NewLimit(limit),
		limitSave: chans.NewLimit(limit),
		m:         m,
		wg:        &sync.WaitGroup{},
	}
}

type PullTrade struct {
	Dir       string
	chanGet   chan *db.Message
	chanSave  chan *db.Message
	limitGet  *chans.Limit
	limitSave *chans.Limit
	m         *tdx.Manage
	wg        *sync.WaitGroup
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
	select {
	case <-ctx.Done():
		return ctx.Err()

	default:

		//1. 获取所有股票代码
		codes := this.m.Codes.GetStocks()

		//2. 获取每只股票的最后数据,加入缓存
		go this.ReadLastFromDB(ctx, codes, this.limitGet, this.chanGet)

		//3. 从服务器拉取数据
		go this.PullData(ctx, this.m, this.chanGet, this.chanSave)

		//4. 更新到数据库
		go this.SaveToDB(ctx, this.limitSave, this.chanSave)

		//5. 等待全部执行完成
		this.wg.Wait()

	}

	return nil
}

func (this *PullTrade) ReadLastFromDB(ctx context.Context, codes []string, limit *chans.Limit, ch chan *db.Message) {
	for i := range codes {
		select {
		case <-ctx.Done():
			return

		default:
			limit.Add()
			go func(code string) {
				defer limit.Done()
				last, err := db.Open(filepath.Join(this.Dir, code+".db")).GetLast()
				if err != nil {
					logs.Err(err)
					return
				}
				ch <- &db.Message{
					Code:  code,
					Trade: last,
				}
				this.wg.Add(1)
			}(codes[i])

		}
	}
}

func (this *PullTrade) PullData(ctx context.Context, m *tdx.Manage, chanGet chan *db.Message, chanSave chan *db.Message) {
	for {
		select {
		case <-ctx.Done():
			return

		case data := <-chanGet:

			if data.Updated() {
				this.wg.Done()
				continue
			}

			m.Go(func(c *tdx.Client) {

				data.RangeDate(func(date string) {
					c.GetHistoryMinuteTradeAll(date, data.Code)

				})

			})
		}
	}
}

func (this *PullTrade) SaveToDB(ctx context.Context, limit *chans.Limit, ch chan *db.Message) {
	for {
		select {
		case <-ctx.Done():
			return

		case data := <-ch:
			limit.Add()
			go func() {
				defer limit.Done()
				defer this.wg.Done()
				_ = data
			}()

		}
	}
}
