package task

import (
	"context"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/tdx"
	"path/filepath"
	"pull-tdx/db"
	"pull-tdx/model"
	"time"
	"xorm.io/xorm"
)

func NewPullTrade(codes []string, dir string, limit int) *PullTrade {
	return &PullTrade{
		Dir:   tradeDir(dir),
		Codes: codes,
		limit: limit,
	}
}

type PullTrade struct {
	Dir   tradeDir //数据保存目录
	Codes []string //用户指定操作的股票
	limit int
}

func (this *PullTrade) Name() string {
	return "更新交易数据"
}

func (this *PullTrade) Run(ctx context.Context, m *tdx.Manage) error {
	r := &Range[string]{
		Codes:   GetCodes(m, this.Codes),
		Append:  nil,
		Limit:   this.limit,
		Retry:   DefaultRetry,
		Handler: this,
	}
	return r.Run(ctx, m)
}

func (this *PullTrade) Handler(ctx context.Context, m *tdx.Manage, code string) error {
	//查询月K线,获取实际上市年份
	firstYear, firstMonth, err := this.getPublic(ctx, m, code)
	if err != nil {
		return err
	}
	return this._range(code, func(year int, filename string) error {
		if year < firstYear {
			return nil
		}

		//1. 打开数据库
		b, err := db.Open(filename)
		if err != nil {
			return err
		}
		defer b.Close()
		b.Sync2(new(model.Trade))

		last, err := b.GetLastTrade()
		if err != nil {
			return err
		}

		if last.Time != 0 && last.Time != 900 {
			//如果最后时间不是15:00,说明数据不全,删除这天的数据
			if _, err := b.Where("Date=?", last.Date).Delete(&model.Trade{}); err != nil {
				return err
			}
			last.Date -= 1
		}

		if last.Date == 0 {
			//说明数据不存在,取该股上市月初为起始时间
			last.Date, _ = model.FromTime(time.Date(year, firstMonth, 1, 0, 0, 0, 0, time.Local))
		}

		//解析日期
		now := time.Now()
		yearLast := time.Date(year, 12, 31, 23, 0, 0, 0, time.Local)
		t := model.ToTime(last.Date, 0)

		//遍历时间,并加入数据库
		for start := t.Add(time.Hour * 24); start.Before(yearLast) && start.Before(now); start = start.Add(time.Hour * 24) {

			//排除休息日
			if !m.Workday.Is(start) {
				continue
			}

			//3. 获取数据
			var insert []*model.Trade
			err = m.Do(func(c *tdx.Client) error {
				//拉取数据
				insert, err = this.pullDay(c, code, start)
				return err
			})
			if err != nil {
				return err
			}

			//排除数据为0的,可能这天停牌了啥的
			if len(insert) == 0 {
				continue
			}
			//插入数据库
			err = b.SessionFunc(func(session *xorm.Session) error {
				for _, v := range insert {
					if _, err := session.Insert(v); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

		}

		return nil
	})

}

// 遍历年份
func (this *PullTrade) _range(code string, fn func(year int, filename string) error) error {
	now := time.Now().Year()
	start := 2000
	for i := start; i <= now; i++ {
		if !oss.Exists(this.Dir.filename(code, i+1)) {
			if err := fn(i, this.Dir.filename(code, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

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

	switch date {
	case 0:
		//

	default:
		//获取历史数据
		resp, err := c.GetHistoryMinuteTradeAll(start.Format("20060102"), code)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			_, minute := model.FromTime(v.Time)
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

func (this tradeDir) lastYear(code string) (year int, filename string) {
	this.rangeYear(code, func(_year int, _filename string) (bool, error) {
		year = _year
		filename = _filename
		return false, nil
	})
	return
}
