package main

import (
	"errors"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/ios/client"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"github.com/robfig/cron/v3"
	"time"
	"xorm.io/xorm"
)

func DialCodes(filename string, op ...client.Option) (*Codes, error) {
	c, err := tdx.DialDefault(op...)
	if err != nil {
		return nil, err
	}
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return nil, err
	}
	if err = db.Sync2(new(Code), new(Update)); err != nil {
		return nil, err
	}
	return NewCodes(c, db)
}

func NewCodes(c *tdx.Client, db *xorms.Engine) (*Codes, error) {
	cs := &Codes{
		c:   c,
		db:  db,
		key: "codes",
	}

	err := cs.Update()
	if err != nil {
		return nil, err
	}

	// 定时更新
	cr := cron.New(cron.WithSeconds())
	_, err = cr.AddFunc("10 0 9 * * *", func() {
		for i := 0; i < 3; i++ {
			if err := cs.Update(); err != nil {
				logs.Err(err)
				<-time.After(time.Minute * 5)
			} else {
				break
			}
		}
	})
	if err != nil {
		return nil, err
	}

	cr.Start()

	return cs, nil
}

type Codes struct {
	c      *tdx.Client   //
	db     *xorms.Engine //
	stocks []string      //缓存
	etfs   []string      //缓存
	key    string        //标识
}

func (this *Codes) GetStocks() []string {
	return this.stocks
}

func (this *Codes) GetEtfs() []string {
	return this.etfs
}

func (this *Codes) updated() (bool, error) {
	update := new(Update)
	{ //查询或者插入一条数据
		has, err := this.db.Where("`Key`=?", this.key).Get(update)
		if err != nil {
			return true, err
		} else if !has {
			update.Key = this.key
			if _, err = this.db.Insert(update); err != nil {
				return true, err
			}
			return false, nil
		}
	}
	{ //判断是否更新过,更新过则不更新
		now := time.Now()
		node := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)
		updateTime := time.Unix(update.Time, 0)
		if now.Sub(node) > 0 {
			//当前时间在9点之后,且更新时间在9点之前,需要更新
			if updateTime.Sub(node) < 0 {
				return false, nil
			}
		} else {
			//当前时间在9点之前,且更新时间在上个节点之前
			if updateTime.Sub(node.Add(time.Hour*24)) < 0 {
				return false, nil
			}
		}
	}
	return true, nil
}

func (this *Codes) Update() error {

	codes, err := this.update()
	if err != nil {
		return err
	}

	stocks := []string(nil)
	etfs := []string(nil)
	for _, v := range codes {
		fullCode := v.FullCode()
		switch {
		case protocol.IsStock(fullCode):
			stocks = append(stocks, fullCode)
		case protocol.IsETF(fullCode):
			etfs = append(etfs, fullCode)
		}
	}

	this.stocks = stocks
	this.etfs = etfs

	return nil
}

// GetCodes 更新股票并返回结果
func (this *Codes) update() ([]*Code, error) {

	if this.c == nil {
		return nil, errors.New("client is nil")
	}

	//2. 查询数据库所有股票
	list := []*Code(nil)
	if err := this.db.Find(&list); err != nil {
		return nil, err
	}

	//如果更新过,则不更新
	updated, err := this.updated()
	if err == nil && updated {
		return list, nil
	}

	mCode := make(map[string]*Code, len(list))
	for _, v := range list {
		mCode[v.FullCode()] = v
	}

	//3. 从服务器获取所有股票代码
	insert := []*Code(nil)
	update := []*Code(nil)
	for _, exchange := range []protocol.Exchange{protocol.ExchangeSH, protocol.ExchangeSZ, protocol.ExchangeBJ} {
		resp, err := this.c.GetCodeAll(exchange)
		if err != nil {
			return nil, err
		}
		for _, v := range resp.List {
			code := &Code{
				Name:      v.Name,
				Code:      v.Code,
				Exchange:  exchange.String(),
				Multiple:  v.Multiple,
				Decimal:   v.Decimal,
				LastPrice: v.LastPrice,
			}
			if val, ok := mCode[exchange.String()+v.Code]; ok {
				if val.Name != v.Name {
					update = append(update, code)
				}
				delete(mCode, exchange.String()+v.Code)
			} else {
				insert = append(insert, code)
				list = append(list, code)
			}
		}
	}

	//4. 插入或者更新数据库
	err = this.db.SessionFunc(func(session *xorm.Session) error {
		for _, v := range mCode {
			if _, err = session.Where("Exchange=? and Code=? ", v.Exchange, v.Code).Delete(v); err != nil {
				return err
			}
		}
		for _, v := range insert {
			if _, err := session.Insert(v); err != nil {
				return err
			}
		}
		for _, v := range update {
			if _, err = session.Where("Exchange=? and Code=? ", v.Exchange, v.Code).Cols("Name,LastPrice").Update(v); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	//更新时间
	_, err = this.db.Where("`Key`=?", this.key).Update(&Update{Time: time.Now().Unix()})
	return list, err
}
