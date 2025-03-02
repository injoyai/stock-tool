package main

import (
	"context"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/conv/cfg/v2"
	inin "github.com/injoyai/goutil/frame/in/v3"
	"github.com/injoyai/goutil/frame/mux"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/robfig/cron/v3"
	"log"
	"path/filepath"
	"pull-minute-trade/plugins"
	"strings"
)

var (
	dir      = cfg.GetString("dir", "./data")
	database = filepath.Join(dir, "database/tdx/")
	config   = &tdx.ManageConfig{
		Hosts:  cfg.GetStrings("hosts"),
		Number: cfg.GetInt("number", 1),
		Dir:    dir,
	}
	disks   = cfg.GetInt("disks", 1)
	spec    = cfg.GetString("spec", "0 1 15 * * *")
	codes   = cfg.GetStrings("codes")
	startup = cfg.GetBool("startup")
)

func init() {
	logs.DefaultFormatter.SetFlag(log.Ltime | log.Lshortfile)
	//logs.SetFormatter(logs.TimeFormatter)
}

func main() {

	logs.Debug(strings.Join(codes, "\n"))

	//1. 连接服务器
	m, err := tdx.NewManage(config)
	logs.PanicErr(err)

	/*



	 */

	//2. 初始化
	queue := chans.NewLimit(1)
	f := func() {
		queue.Add()
		defer queue.Done()
		pull := plugins.NewPullTrade(m, codes, database, disks)
		err = pull.Run(context.Background())
		logs.Infof("任务[%s]完成, 结果: %v\n", pull.Name(), conv.New(err).String("成功"))
	}

	//3. 设置定时
	cr := cron.New(cron.WithSeconds())
	cr.AddFunc(spec, f)
	cr.Start()

	//4. 启动便执行
	if startup {
		logs.Info("马上执行...")
		go f()
	}

	/*



	 */

	//5. 开启HTTP服务
	s := mux.New()
	s.Group("/api", func(g *mux.Grouper) {
		g.POST("/task", func(r *mux.Request) {

		})
		g.POST("/execute", func(r *mux.Request) {
			if !queue.Try() {
				inin.Fail("有任务正在执行")
			}
			defer queue.Done()

		})
	})
	s.SetPort(20001).Run()

}
