package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/injoyai/conv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
)

func DialCodesHTTP(address string, spec ...string) (c *CodesHTTP, err error) {
	c = &CodesHTTP{address: address, CodesBase: tdx.NewCodesBase()}
	cr := cron.New(cron.WithSeconds())
	_spec := conv.Default("0 20 9 * * *", spec...)
	_, err = cr.AddFunc(_spec, func() { logs.PrintErr(c.Update()) })
	if err != nil {
		return
	}
	err = c.Update()
	if err != nil {
		return
	}
	cr.Start()
	return c, nil
}

type CodesHTTP struct {
	address string
	*tdx.CodesBase
}

func (this *CodesHTTP) Update() error {
	ls, err := this.getList("/all")
	if err != nil {
		return err
	}
	this.CodesBase.Update(ls)
	return nil
}

func (this *CodesHTTP) getList(path string) (tdx.CodeModels, error) {
	resp, err := http.DefaultClient.Get(this.address + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http code:%d", resp.StatusCode)
	}
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ls := tdx.CodeModels{}
	err = json.Unmarshal(bs, &ls)
	return ls, err
}
