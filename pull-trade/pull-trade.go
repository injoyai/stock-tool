package main

import (
	"context"
	"fmt"
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

func NewSqlite(codes []string, _dir string, limit, tasks int) *Sqlite {
	tasks = conv.Select(tasks < 1, 2, tasks)
	return &Sqlite{
		Dir:       dir(_dir),
		Codes:     codes,
		limit:     limit,
		tasks:     tasks,
		Chan:      make(chan func(), limit+1),
		StartDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.Local),
	}
}

type Sqlite struct {
	Dir       dir         //数据保存目录
	Codes     []string    //用户指定操作的股票
	limit     int         //协程数量
	tasks     int         //Tasks 每次任务数量
	Chan      chan func() //队列插入
	StartDate time.Time   //最早日期
}

func (this *Sqlite) Name() string {
	return "更新交易数据"
}

func (this *Sqlite) Run(ctx context.Context, m *tdx.Manage) error {
	codes := GetCodes(m, this.Codes)
	tasks := this.tasks
	var cs []string
	for offset := 0; ; offset += tasks {
		if offset >= len(codes) {
			logs.Debug("read done")
			break
		}
		if offset+tasks > len(codes) {
			cs = codes[offset:]
		} else {
			cs = codes[offset : offset+tasks]
		}

		logs.Debugf("1. 读取任务: %d*year\n", tasks)
		ls := this.readAll(ctx, m, cs)
		logs.Debug("2. 任务数量:", len(ls))

		//此次任务结束信号
		pullDone := make(chan struct{}, 1)
		go this.pull(ctx, m, ls, pullDone)

		logs.Debug("3. 写入硬盘")
		this.commit(ctx, len(ls), pullDone)
	}

	return nil
}

func (this *Sqlite) pull(ctx context.Context, m *tdx.Manage, dbs []*tradeDB, pullDone chan struct{}) {
	limit := chans.NewWaitLimit(this.limit)
	for _, b := range dbs {
		limit.Add()
		go func(b *tradeDB) {
			defer limit.Done()
			err := g.Retry(func() error { return this._pull(ctx, b, m) }, DefaultRetry)
			logs.PrintErr(err)
		}(b)
	}
	go func() {
		limit.Wait()
		close(pullDone)
	}()
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
	var public time.Time
	once := sync.Once{}

	ls := []*tradeDB(nil)
	now := time.Now()
	err := this.Dir.rangeYear(code, func(year int, filename string, exist, hasNext bool) (bool, error) {

		//存在,并且不是今年,不需要更新
		//如果是今年,则需要实时更新,
		//例如跨年的时候,有可能需要补充去年的数据
		if exist && year < now.Year() && hasNext {
			return true, nil
		}

		var err error
		once.Do(func() {
			//查询月K线,获取实际上市年份
			public, err = this.getPublic(m, code)
		})
		if err != nil {
			return false, err
		}

		//年份小于上市年份,无效,跳过
		if year < public.Year() {
			return true, nil
		}

		x, err := newTradeDB(filename, code, year, public)
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
					logs.Debug("commit done")
					return nil
				}
			}

		}
	}
	return nil
}

func (this *Sqlite) _pull(ctx context.Context, b *tradeDB, m *tdx.Manage) (err error) {
	defer b.CloseWithErr(err)

	now := time.Now()

	defer func() {
		logs.Debugf("[%s-%d] 插入耗时: %s\n", b.Code, b.Year, time.Since(now))
	}()

	yearLast := time.Date(b.Year, 12, 31, 23, 0, 0, 0, time.Local)
	t := ToTime(b.LastDate, 0)

	var insert []*Trade
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

			logs.Debug(b.Code, b.Year, date.Format("2006-01-02"))

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

	logs.Debugf("[%s-%d] 拉取耗时: %s 数量: %d\n", b.Code, b.Year, time.Since(now), len(insert))

	if len(insert) == 0 {
		return nil
	}

	//初始化操作,例如新建文件,表等
	if err = b.init(); err != nil {
		return err
	}
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

	//写入硬盘,有单线程统一处理,顺序写入
	this.Chan <- func() {
		err = session.Commit()
		if err != nil {
			//发生错误,删除文件
			b.CloseWithErr(err)
		}
		//无错误,是否资源
		defer b.Close()
		session.Close()
	}

	return nil

}

// 获取上市年月
func (this *Sqlite) getPublic(m *tdx.Manage, code string) (public time.Time, err error) {
	year := 1990
	month := time.Month(12)
	err = m.Do(func(c *tdx.Client) error {
		resp, err := c.GetKlineMonthAll(code)
		if err != nil {
			return err
		}
		if len(resp.List) == 0 {
			return fmt.Errorf("股票[%s]可能已经退市", code)
		}
		if len(resp.List) > 0 {
			year = resp.List[0].Time.Year()
			month = resp.List[0].Time.Month()
			return nil
		}
		return nil
	})
	public = time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	return
}

// pullDay 按天拉取数据
func (this *Sqlite) pullDay(c *tdx.Client, code string, start time.Time) ([]*Trade, error) {

	insert := []*Trade(nil)

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
			insert = append(insert, &Trade{
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
		resp, err := c.GetHistoryTradeDay(start.Format("20060102"), code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := FromTime(v.Time)
			insert = append(insert, &Trade{
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

type dir string

func (this dir) filename(code string, year int) string {
	return filepath.Join(string(this), code, code+"-"+conv.String(year)+".db")
}

// 遍历年份,返回未完成的年份和文件名称
func (this dir) rangeYear(code string, fn func(year int, filename string, exist, hasNext bool) (bool, error)) error {
	now := time.Now().Year()
	start := 2000
	for i := start; i <= now; i++ {
		filename := this.filename(code, i)
		if oss.Exists(filename + "-journal") {
			//说明这个文件的数据不全
			logs.Trace("删除:", filename)
			if err := os.Remove(filename); err != nil {
				return err
			}
			logs.Trace("删除:", filename+"-journal")
			if err := os.Remove(filename + "-journal"); err != nil {
				return err
			}
		}
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

func newTradeDB(filename, code string, year int, public time.Time) (*tradeDB, error) {
	t := &tradeDB{
		Code:     code,
		Filename: filename,
		Year:     year,
		Public:   public,
	}
	//不存在该年的数据,时间从该年的1.1开始
	t.LastDate, _ = FromTime(time.Date(year, 1, 1, 0, 0, 0, 0, time.Local))
	if !oss.Exists(filename) {
		return t, nil
	}
	err := t.init()
	return t, err
}

type tradeDB struct {
	Code     string        //代码
	Filename string        //文件名称
	DB       *xorms.Engine //数据库实例

	LastDate uint16    //最后数据日期
	Year     int       //任务的年份
	Public   time.Time //上市时间
}

func (this *tradeDB) init() (err error) {
	if this.DB == nil {
		exist := oss.Exists(this.Filename)

		this.DB, err = sqlite.NewXorm(this.Filename)
		if err != nil {
			return err
		}
		if err = this.DB.Sync2(new(Trade)); err != nil {
			return err
		}

		last := new(Trade)
		if exist {
			_, err = this.DB.Desc("Date", "Time").Get(last)
			if err != nil {
				return err
			}
			if last.Time != 0 && last.Time != 900 && last.Time != 899 {
				//如果最后时间不是15:00/14:59(早期),说明数据不全,删除这天的数据
				if _, err = this.DB.Where("Date=?", last.Date).Delete(&Trade{}); err != nil {
					return err
				}
				last.Date -= 1
			}
		}

		this.LastDate = last.Date
		if this.LastDate == 0 {
			//说明数据不存在,取该股上市月初为起始时间
			month := conv.Select(this.Year == this.Public.Year(), this.Public.Month(), 1)
			this.LastDate, _ = FromTime(time.Date(this.Year, month, 1, 0, 0, 0, 0, time.Local))
		}

	}
	return
}

func (this *tradeDB) Close() {
	this.DB.Close()
}

func (this *tradeDB) CloseWithErr(err error) error {
	if err == nil {
		return nil
	}
	if this.DB != nil {
		this.DB.Close()
	}
	return os.Remove(this.Filename)
}
