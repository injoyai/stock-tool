package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func NewSqlite(codes []string, dir string, limit int) *Sqlite {
	return &Sqlite{
		Dir:       tradeDir(dir),
		Codes:     codes,
		limit:     limit,
		Chan:      make(chan func(), 100),
		StartDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local),
	}
}

type Sqlite struct {
	Dir       tradeDir    //数据保存目录
	Codes     []string    //用户指定操作的股票
	limit     int         //最大并发,HHD推荐1
	Chan      chan func() //队列插入
	StartDate time.Time   //最早日期
}

func (this *Sqlite) Name() string {
	return "更新交易数据"
}

func (this *Sqlite) Run(ctx context.Context, m *tdx.Manage) error {

	codes := GetCodes(m, this.Codes)
	dbs := make(chan *tradeDB, 1000)
	readDone := make(chan struct{}, 1)
	pullDone := make(chan struct{}, 1)

	go func() {
		limit := chans.NewLimit(this.limit)
		logs.Debug("limit:", this.limit)
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
					logs.Debug("read done")
					close(pullDone)
					return
				default:
				}
			}
		}
	}()

	limit := 800
	var cs []string
	for offset := 0; ; offset += limit {
		if offset >= len(codes) {
			close(readDone)
			break
		}
		if offset+limit > len(codes) {
			cs = codes[offset:]
		} else {
			cs = codes[offset : offset+limit]
		}
		logs.Debug("readAll")
		ls := this.readAll(ctx, m, cs)
		for _, v := range ls {
			dbs <- v
		}
		logs.Debug("commit")
		this.commit(ctx, len(ls), pullDone)
	}

	return nil
}

func (this *Sqlite) readAll(ctx context.Context, m *tdx.Manage, codes []string) []*tradeDB {
	lss := []*tradeDB(nil)
	limit := chans.NewWaitLimit(this.limit)
	mu := sync.Mutex{}
	for _, v := range codes {
		limit.Add()
		go func(v string) {
			defer limit.Done()
			err := g.Retry(func() error {
				ls, err := this.readOne(ctx, m, v)
				if err != nil {
					return err
				}
				mu.Lock()
				defer mu.Unlock()
				lss = append(lss, ls...)
				return nil
			}, DefaultRetry)
			logs.PrintErr(err)
		}(v)
	}
	limit.Wait()
	return lss
}

func (this *Sqlite) readOne(ctx context.Context, m *tdx.Manage, code string) ([]*tradeDB, error) {
	//查询月K线,获取实际上市年份
	publicYear, publicMonth, err := this.getPublic(ctx, m, code)
	if err != nil {
		return nil, err
	}

	ls := []*tradeDB(nil)
	now := time.Now()
	err = this.Dir.rangeYearAll(code, func(year int, filename string, exist, hasNext bool) (bool, error) {
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
		ls = append(ls, x)
		return true, nil
	})
	return ls, err
}

func (this *Sqlite) commit(ctx context.Context, num int, done chan struct{}) error {
	for i := 0; i < num; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case fn, ok := <-this.Chan:

			if !ok {
				return nil
			}
			now := time.Now()
			fn()
			logs.Debugf("序号: %d 排队: %d 耗时: %s\n", i, len(this.Chan), time.Since(now))

		case <-done:
			for {
				select {
				case fn, ok := <-this.Chan:
					if !ok {
						return nil
					}
					fn()
				default:
					logs.Debug("commit done")
					return nil
				}
			}

		}
	}
	return nil
}

func (this *Sqlite) pull(ctx context.Context, b *tradeDB, m *tdx.Manage) (err error) {
	defer b.CloseWithErr(err)

	now := time.Now()

	defer func() {
		logs.Debugf("[%s-%d] 插入耗时: %s\n", b.Code, b.Year, time.Since(now))
	}()

	yearLast := time.Date(b.Year, 12, 31, 23, 0, 0, 0, time.Local)
	t := ToTime(b.Last.Date, 0)

	var insert []*TradeSqlite
	err = m.Do(func(c *tdx.Client) error {
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
			item, err := this.pullDay(c, b.Code, date)
			if err != nil {
				return err
			}
			insert = append(insert, item...)
		}
		return nil
	})
	if err != nil {
		return err
	}
	logs.Debugf("[%s-%d] 拉取耗时: %s\n 数量: %d\n", b.Code, b.Year, time.Since(now), len(insert))
	session := b.DB.Engine.NewSession()
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

// 获取上市年月
func (this *Sqlite) getPublic(ctx context.Context, m *tdx.Manage, code string) (year int, month time.Month, err error) {
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
func (this *Sqlite) pullDay(c *tdx.Client, code string, start time.Time) ([]*TradeSqlite, error) {

	insert := []*TradeSqlite(nil)

	date, _ := FromTime(start)

	nowDate, _ := FromTime(time.Now())

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
			_, minute := FromTime(v.Time)
			insert = append(insert, &TradeSqlite{
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
			_, minute := FromTime(v.Time)
			insert = append(insert, &TradeSqlite{
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
	b, err := sqlite.NewXorm(filename)
	if err != nil {
		return nil, err
	}
	b.Sync2(new(TradeSqlite))

	last := new(TradeSqlite)
	_, err = b.Desc("Date", "Time").Get(last)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	if last.Time != 0 && last.Time != 900 {
		//如果最后时间不是15:00,说明数据不全,删除这天的数据
		if _, err := b.Where("Date=?", last.Date).Delete(&TradeSqlite{}); err != nil {
			return nil, err
		}
		last.Date -= 1
	}

	if last.Date == 0 {
		//说明数据不存在,取该股上市月初为起始时间
		month := conv.Select(year == publicYear, publicMonth, 1)
		last.Date, _ = FromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.Local))
	}
	return &tradeDB{
		Code:        code,
		Filename:    filename,
		DB:          b,
		Last:        last,
		Year:        year,
		PublicYear:  publicYear,
		PublicMonth: publicMonth,
	}, nil
}

type tradeDB struct {
	Code        string
	Filename    string
	DB          *xorms.Engine
	Last        *TradeSqlite
	Year        int
	PublicYear  int
	PublicMonth time.Month
}

func (this *tradeDB) Close() {
	this.DB.Close()
}

func (this *tradeDB) CloseWithErr(err error) error {
	if err == nil {
		return nil
	}
	this.DB.Close()
	return os.Remove(this.Filename)
}
