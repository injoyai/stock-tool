package task

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/base/types"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"
	"time"
)

func NewPullTrade(codes []string, dir string, limit int) *PullTrade {
	return &PullTrade{
		Dir:       tradeDir(dir),
		Codes:     codes,
		limit:     limit,
		Chan:      make(chan func(), 100),
		StartDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local),
	}
}

type PullTrade struct {
	Dir       tradeDir    //数据保存目录
	Codes     []string    //用户指定操作的股票
	limit     int         //最大并发,HHD推荐1
	Chan      chan func() //队列插入
	StartDate time.Time   //最早日期
}

func (this *PullTrade) Name() string {
	return "更新交易数据"
}

func (this *PullTrade) Run(ctx context.Context, m *tdx.Manage) error {

	codes := types.List[string](GetCodes(m, this.Codes))
	dbs := make(chan *tradeDB, 1000)
	readDone := make(chan struct{}, 1)
	pullDone := make(chan struct{}, 1)

	go func() {
		limit := chans.NewLimit(this.limit)
		for {
			select {
			case <-ctx.Done():
				return
			case <-readDone:
			case b := <-dbs:
				limit.Add()
				go func(b *tradeDB) {
					defer limit.Done()
					err := g.Retry(func() error { return this.pull(ctx, b, m) }, DefaultRetry)
					logs.PrintErr(err)
				}(b)

			default:
				select {
				case <-readDone:
					close(pullDone)
					return
				default:
				}
			}
		}
	}()

	limit := 800
	for offset := 0; ; offset += limit {
		cs := codes.Cut(offset, limit)
		this.readAll(ctx, m, cs, dbs)
		if len(cs) == 0 {
			close(readDone)
			break
		}
		this.commit(ctx, pullDone)
	}

	return nil
}

func (this *PullTrade) readAll(ctx context.Context, m *tdx.Manage, codes []string, ch chan *tradeDB) {
	limit := chans.NewLimit(this.limit)
	for _, v := range codes {
		limit.Add()
		go func(v string) {
			defer limit.Done()
			err := g.Retry(func() error { return this.readOne(ctx, m, v, ch) }, DefaultRetry)
			logs.PrintErr(err)
		}(v)
	}
}

func (this *PullTrade) readOne(ctx context.Context, m *tdx.Manage, code string, ch chan *tradeDB) error {
	//查询月K线,获取实际上市年份
	publicYear, publicMonth, err := this.getPublic(ctx, m, code)
	if err != nil {
		return err
	}

	now := time.Now()
	return this.Dir.rangeYearAll(code, func(year int, filename string, exist, hasNext bool) (bool, error) {
		//存在,并且不是今年,今年存在并需要实时更新
		if exist && year < now.Year() && !hasNext {
			return true, nil
		}

		//年份小于上市年份,无效,跳过
		if year < publicYear {
			return true, nil
		}

		x, err := newTradeDB(filename, code, year, publicYear, publicMonth)
		if err != nil {
			return true, err
		}
		ch <- x
		return true, nil
	})
}

func (this *PullTrade) commit(ctx context.Context, done chan struct{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case fn, ok := <-this.Chan:
			if !ok {
				return nil
			}
			fn()

		case <-done:
			for {
				select {
				case fn, ok := <-this.Chan:
					if !ok {
						return nil
					}
					fn()
				default:
					return nil
				}
			}

		}
	}
}

func (this *PullTrade) pull(ctx context.Context, b *tradeDB, m *tdx.Manage) error {
	now := time.Now()
	yearLast := time.Date(b.Year, 12, 31, 23, 0, 0, 0, time.Local)
	t := model.ToTime(b.Last.Date, 0)

	var insert []*model.Trade

	//遍历时间,拉取数据并加入数据库
	for date := t.Add(time.Hour * 24); date.Before(yearLast) && date.Before(now); date = date.Add(time.Hour * 24) {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		//最早日期为2000-06-09
		if date.Before(this.StartDate) {
			continue
		}

		//排除休息日
		if !m.Workday.Is(date) {
			continue
		}

		//3. 获取数据
		err := m.Do(func(c *tdx.Client) error {
			//拉取数据
			item, err := this.pullDay(c, b.Code, date)
			if err != nil {
				return err
			}
			insert = append(insert, item...)
			return nil
		})
		if err != nil {
			return err
		}

	}

	session := b.DB.NewSession()
	if err := session.Begin(); err != nil {
		session.Close()
		return err
	}

	for _, v := range insert {
		if _, err := session.Insert(v); err != nil {
			session.Close()
			return err
		}
	}

	this.Chan <- func() {
		defer b.Close()
		session.Commit()
		session.Close()
	}

	return nil

}

//func (this *PullTrade) Handler(ctx context.Context, m *tdx.Manage, code string, doneItem func()) error {
//
//	//查询月K线,获取实际上市年份
//	firstYear, firstMonth, err := this.getPublic(ctx, m, code)
//	if err != nil {
//		return err
//	}
//
//	return this.Dir.rangeYearAll(code, func(year int, filename string, hasNext bool) (bool, error) {
//		defer doneItem()
//
//		//存在后一年的数据,跳过
//		if hasNext {
//			return true, nil
//		}
//
//		//年份小于上市年份,跳过
//		if year < firstYear {
//			return true, nil
//		}
//
//		//1. 打开数据库
//		b, err := db.Open(filename)
//		if err != nil {
//			return false, err
//		}
//		defer b.Close()
//		b.Sync2(new(model.Trade))
//
//		last, err := b.GetLastTrade()
//		if err != nil {
//			return false, err
//		}
//
//		if last.Time != 0 && last.Time != 900 {
//			//如果最后时间不是15:00,说明数据不全,删除这天的数据
//			if _, err := b.Where("Date=?", last.Date).Delete(&model.Trade{}); err != nil {
//				return false, err
//			}
//			last.Date -= 1
//		}
//
//		if last.Date == 0 {
//			//说明数据不存在,取该股上市月初为起始时间
//			month := conv.Select(year == firstYear, firstMonth, 1)
//			last.Date, _ = model.FromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.Local))
//		}
//		//logs.Debug("开始日期:", model.ToTime(last.Date, 0).Format("2006-01-02"))
//
//		//解析日期
//		now := time.Now()
//		yearLast := time.Date(year, 12, 31, 23, 0, 0, 0, time.Local)
//		t := model.ToTime(last.Date, 0)
//
//		//遍历时间,拉取数据并加入数据库
//		for date := t.Add(time.Hour * 24); date.Before(yearLast) && date.Before(now); date = date.Add(time.Hour * 24) {
//
//			//最早日期为2000-06-09
//			if date.Before(time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local)) {
//				continue
//			}
//
//			//排除休息日
//			if !m.Workday.Is(date) {
//				continue
//			}
//
//			//3. 获取数据
//			var insert []*model.Trade
//			err = m.Do(func(c *tdx.Client) error {
//				//拉取数据
//				insert, err = this.pullDay(c, code, date)
//				return err
//			})
//			if err != nil {
//				return false, err
//			}
//
//			//排除数据为0的,可能这天停牌了啥的
//			if len(insert) == 0 {
//				continue
//			}
//			//插入数据库
//			err = b.SessionFunc(func(session *xorm.Session) error {
//				for _, v := range insert {
//					if _, err := session.Insert(v); err != nil {
//						return err
//					}
//				}
//				return nil
//			})
//			if err != nil {
//				return false, err
//			}
//
//		}
//
//		return true, nil
//	})
//
//}

// 获取上市年月
func (this *PullTrade) getPublic(ctx context.Context, m *tdx.Manage, code string) (year int, month time.Month, err error) {
	year = 1990
	month = 12
	err = m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineMonthAll(code)
		if err != nil {
			return err
		}
		if len(resp.List) > 0 {
			year = resp.List[0].Time.Year()
			month = resp.List[0].Time.Month()
			return nil
		}
		return nil
	})
	return
}

// pullDay 按天拉取数据
func (this *PullTrade) pullDay(c *tdx.Client, code string, start time.Time) ([]*model.Trade, error) {

	insert := []*model.Trade(nil)

	date, _ := model.FromTime(start)

	nowDate, _ := model.FromTime(time.Now())

	switch date {
	case 0:
	//

	case nowDate:
		//获取当天数据
		resp, err := c.GetTradeAll(code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := model.FromTime(v.Time)
			insert = append(insert, &model.Trade{
				Date:   date,
				Time:   minute,
				Price:  v.Price,
				Volume: v.Volume,
				Order:  v.Number,
				Status: v.Status,
			})
		}

	default:
		//获取历史数据
		resp, err := c.GetHistoryTradeAll(start.Format("20060102"), code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := model.FromTime(v.Time)
			insert = append(insert, &model.Trade{
				Date:   date,
				Time:   minute,
				Price:  v.Price,
				Volume: v.Volume,
				Order:  0,
				Status: v.Status,
			})
		}

	}

	return insert, nil
}

type tradeDir string

func (this tradeDir) filename(code string, year int) string {
	return filepath.Join(string(this), code, code+"-"+conv.String(year)+".db")
}

// 遍历年份,返回未完成的年份和文件名称
func (this tradeDir) rangeYear(code string, fn func(year int, filename string) (bool, error)) error {
	now := time.Now().Year()
	start := 2000
	for i := start; i <= now; i++ {
		filename := this.filename(code, i+1)
		if !oss.Exists(filename) {
			next, err := fn(i, this.filename(code, i))
			if err != nil {
				return err
			}
			if !next {
				break
			}
		}
	}
	return nil
}

// 遍历年份,返回未完成的年份和文件名称
func (this tradeDir) rangeYearAll(code string, fn func(year int, filename string, exist, hasNext bool) (bool, error)) error {
	now := time.Now().Year()
	start := 2000
	for i := start; i <= now; i++ {
		filename := this.filename(code, i)
		next, err := fn(i, filename, oss.Exists(filename), oss.Exists(this.filename(code, i+1)))
		if err != nil {
			return err
		}
		if !next {
			break
		}
	}
	return nil
}

func (this tradeDir) lastYear(code string) (year int, filename string) {
	this.rangeYear(code, func(_year int, _filename string) (bool, error) {
		year = _year
		filename = _filename
		return true, nil
	})
	return
}

func newTradeDB(filename, code string, year, publicYear int, publicMonth time.Month) (*tradeDB, error) {
	b, err := db.Open(filename)
	if err != nil {
		return nil, err
	}
	b.Sync2(new(model.Trade))
	last, err := b.GetLastTrade()
	if err != nil {
		return nil, err
	}
	if last.Time != 0 && last.Time != 900 {
		//如果最后时间不是15:00,说明数据不全,删除这天的数据
		if _, err := b.Where("Date=?", last.Date).Delete(&model.Trade{}); err != nil {
			return nil, err
		}
		last.Date -= 1
	}

	if last.Date == 0 {
		//说明数据不存在,取该股上市月初为起始时间
		month := conv.Select(year == publicYear, publicMonth, 1)
		last.Date, _ = model.FromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.Local))
	}
	return &tradeDB{
		Code:        code,
		DB:          b,
		Last:        last,
		Year:        year,
		PublicYear:  publicYear,
		PublicMonth: publicMonth,
	}, nil
}

type tradeDB struct {
	Code        string
	DB          *db.Sqlite
	Last        *model.Trade
	Year        int
	PublicYear  int
	PublicMonth time.Month
}

func (this *tradeDB) Close() {
	this.DB.Close()
}
