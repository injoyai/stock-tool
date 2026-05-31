package main

import (
	"path/filepath"
	"time"

	"github.com/injoyai/bar"
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"github.com/injoyai/goutil/database/sqlite"
	"github.com/injoyai/goutil/database/xorms"
	"github.com/injoyai/goutil/g"
	"github.com/injoyai/goutil/oss"
	"github.com/injoyai/goutil/other/csv"
	"github.com/injoyai/logs"
	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
	"xorm.io/xorm"
)

var (
	DatabaseDir   = "./data/database/kline"
	CsvDir        = "./data/csv"
	Clients       = 3
	Goroutine     = 6
	Startup       = true
	Retry         = 3
	RetryInterval = time.Second

	Codes = []string{
		"sh501001",
		"sh501005",
		"sh501006",
		"sh501007",
		"sh501008",
		"sh501009",
		"sh501010",
		"sh501011",
		"sh501012",
		"sh501015",
		"sh501016",
		"sh501017",
		"sh501018",
		"sh501019",
		"sh501021",
		"sh501022",
		"sh501023",
		"sh501025",
		"sh501026",
		"sh501028",
		"sh501029",
		"sh501030",
		"sh501031",
		"sh501032",
		"sh501036",
		"sh501037",
		"sh501038",
		"sh501043",
		"sh501045",
		"sh501046",
		"sh501047",
		"sh501048",
		"sh501050",
		"sh501051",
		"sh501053",
		"sh501057",
		"sh501058",
		"sh501059",
		"sh501060",
		"sh501061",
		"sh501062",
		"sh501064",
		"sh501065",
		"sh501070",
		"sh501071",
		"sh501073",
		"sh501075",
		"sh501076",
		"sh501077",
		"sh501078",
		"sh501079",
		"sh501080",
		"sh501081",
		"sh501082",
		"sh501083",
		"sh501085",
		"sh501087",
		"sh501088",
		"sh501089",
		"sh501090",
		"sh501091",
		"sh501092",
		"sh501093",
		"sh501095",
		"sh501096",
		"sh501097",
		"sh501098",
		"sh501099",
		"sh501186",
		"sh501188",
		"sh501189",
		"sh501200",
		"sh501201",
		"sh501202",
		"sh501203",
		"sh501205",
		"sh501206",
		"sh501207",
		"sh501208",
		"sh501209",
		"sh501210",
		"sh501211",
		"sh501212",
		"sh501213",
		"sh501215",
		"sh501216",
		"sh501217",
		"sh501218",
		"sh501219",
		"sh501220",
		"sh501222",
		"sh501225",
		"sh501227",
		"sh501300",
		"sh501301",
		"sh501302",
		"sh501303",
		"sh501305",
		"sh501306",
		"sh501307",
		"sh501310",
		"sh501311",
		"sh501312",
		"sh502000",
		"sh502003",
		"sh502006",
		"sh502010",
		"sh502013",
		"sh502023",
		"sh502048",
		"sh502053",
		"sh502056",
		"sh506000",
		"sh506001",
		"sh506002",
		"sh506003",
		"sh506005",
		"sh506006",
		"sh506008",
		"sh508000",
		"sh508001",
		"sh508002",
		"sh508003",
		"sh508005",
		"sh508006",
		"sh508007",
		"sh508008",
		"sh508009",
		"sh508010",
		"sh508011",
		"sh508012",
		"sh508015",
		"sh508016",
		"sh508017",
		"sh508018",
		"sh508019",
		"sh508021",
		"sh508022",
		"sh508026",
		"sh508027",
		"sh508028",
		"sh508029",
		"sh508031",
		"sh508033",
		"sh508036",
		"sh508039",
		"sh508048",
		"sh508055",
		"sh508056",
		"sh508058",
		"sh508060",
		"sh508066",
		"sh508068",
		"sh508069",
		"sh508077",
		"sh508078",
		"sh508080",
		"sh508082",
		"sh508084",
		"sh508085",
		"sh508086",
		"sh508087",
		"sh508088",
		"sh508089",
		"sh508090",
		"sh508091",
		"sh508092",
		"sh508096",
		"sh508097",
		"sh508098",
		"sh508099",
		"sh520500",
		"sh520510",
		"sh520520",
		"sh520550",
		"sh520560",
		"sh520570",
		"sh520580",
		"sh520600",
		"sh520620",
		"sh520650",
		"sh520660",
		"sh520670",
		"sh520690",
		"sh520700",
		"sh520720",
		"sh520830",
		"sh520840",
		"sh520860",
		"sh520870",
		"sh520880",
		"sh520890",
		"sh520900",
		"sh520920",
		"sh520940",
		"sh520950",
		"sh520960",
		"sh520970",
		"sh520980",
		"sh520990",
		"sh530000",
		"sh530050",
		"sh530080",
		"sh530100",
		"sh530180",
		"sh530280",
		"sh530300",
		"sh530380",
		"sh530530",
		"sh530580",
		"sh530680",
		"sh530800",
		"sh530880",
		"sh520530",
		"sh520590",
		"sh520630",
		"sh520780",
		"sh520820",
		"sz160105",
		"sz160106",
		"sz160119",
		"sz160125",
		"sz160127",
		"sz160128",
		"sz160133",
		"sz160135",
		"sz160137",
		"sz160140",
		"sz160142",
		"sz160143",
		"sz160211",
		"sz160212",
		"sz160215",
		"sz160216",
		"sz160218",
		"sz160219",
		"sz160220",
		"sz160221",
		"sz160222",
		"sz160223",
		"sz160225",
		"sz160311",
		"sz160314",
		"sz160322",
		"sz160323",
		"sz160324",
		"sz160325",
		"sz160326",
		"sz160416",
		"sz160418",
		"sz160419",
		"sz160420",
		"sz160421",
		"sz160425",
		"sz160505",
		"sz160512",
		"sz160513",
		"sz160515",
		"sz160516",
		"sz160517",
		"sz160518",
		"sz160526",
		"sz160527",
		"sz160529",
		"sz160603",
		"sz160605",
		"sz160607",
		"sz160610",
		"sz160611",
		"sz160613",
		"sz160615",
		"sz160616",
		"sz160617",
		"sz160618",
		"sz160620",
		"sz160621",
		"sz160622",
		"sz160624",
		"sz160625",
		"sz160626",
		"sz160627",
		"sz160628",
		"sz160629",
		"sz160630",
		"sz160631",
		"sz160632",
		"sz160633",
		"sz160634",
		"sz160635",
		"sz160636",
		"sz160637",
		"sz160638",
		"sz160639",
		"sz160641",
		"sz160642",
		"sz160643",
		"sz160644",
		"sz160646",
		"sz160706",
		"sz160716",
		"sz160717",
		"sz160718",
		"sz160719",
		"sz160722",
		"sz160723",
		"sz160726",
		"sz160727",
		"sz160805",
		"sz160806",
		"sz160807",
		"sz160812",
		"sz160813",
		"sz160910",
		"sz160916",
		"sz160918",
		"sz160919",
		"sz160921",
		"sz160924",
		"sz160925",
		"sz160926",
		"sz161005",
		"sz161010",
		"sz161014",
		"sz161015",
		"sz161017",
		"sz161019",
		"sz161022",
		"sz161024",
		"sz161025",
		"sz161026",
		"sz161027",
		"sz161028",
		"sz161029",
		"sz161030",
		"sz161031",
		"sz161032",
		"sz161033",
		"sz161035",
		"sz161036",
		"sz161037",
		"sz161038",
		"sz161039",
		"sz161040",
		"sz161115",
		"sz161116",
		"sz161117",
		"sz161118",
		"sz161119",
		"sz161121",
		"sz161122",
		"sz161123",
		"sz161124",
		"sz161125",
		"sz161126",
		"sz161127",
		"sz161128",
		"sz161129",
		"sz161130",
		"sz161131",
		"sz161132",
		"sz161133",
		"sz161216",
		"sz161217",
		"sz161219",
		"sz161222",
		"sz161224",
		"sz161225",
		"sz161226",
		"sz161227",
		"sz161229",
		"sz161232",
		"sz161233",
		"sz161505",
		"sz161607",
		"sz161610",
		"sz161614",
		"sz161626",
		"sz161628",
		"sz161631",
		"sz161706",
		"sz161713",
		"sz161715",
		"sz161716",
		"sz161720",
		"sz161721",
		"sz161722",
		"sz161723",
		"sz161724",
		"sz161725",
		"sz161726",
		"sz161727",
		"sz161728",
		"sz161729",
		"sz161730",
		"sz161810",
		"sz161811",
		"sz161812",
		"sz161815",
		"sz161816",
		"sz161818",
		"sz161820",
		"sz161831",
		"sz161834",
		"sz161837",
		"sz161838",
		"sz161903",
		"sz161907",
		"sz161908",
		"sz161912",
		"sz161914",
		"sz162006",
		"sz162105",
		"sz162107",
		"sz162108",
		"sz162207",
		"sz162215",
		"sz162216",
		"sz162307",
		"sz162411",
		"sz162412",
		"sz162414",
		"sz162415",
		"sz162509",
		"sz162511",
		"sz162605",
		"sz162607",
		"sz162703",
		"sz162711",
		"sz162712",
		"sz162714",
		"sz162715",
		"sz162719",
		"sz162720",
		"sz162721",
		"sz163001",
		"sz163003",
		"sz163005",
		"sz163109",
		"sz163110",
		"sz163111",
		"sz163113",
		"sz163114",
		"sz163115",
		"sz163116",
		"sz163118",
		"sz163208",
		"sz163209",
		"sz163302",
		"sz163402",
		"sz163406",
		"sz163407",
		"sz163409",
		"sz163412",
		"sz163415",
		"sz163417",
		"sz163418",
		"sz163503",
		"sz163801",
		"sz163803",
		"sz163804",
		"sz163805",
		"sz163806",
		"sz163807",
		"sz163808",
		"sz163809",
		"sz163810",
		"sz163819",
		"sz163821",
		"sz163907",
		"sz164105",
		"sz164206",
		"sz164208",
		"sz164210",
		"sz164212",
		"sz164304",
		"sz164401",
		"sz164402",
		"sz164403",
		"sz164508",
		"sz164509",
		"sz164606",
		"sz164701",
		"sz164703",
		"sz164705",
		"sz164808",
		"sz164810",
		"sz164814",
		"sz164818",
		"sz164824",
		"sz164826",
		"sz164902",
		"sz164905",
		"sz164906",
		"sz164908",
		"sz165309",
		"sz165310",
		"sz165311",
		"sz165312",
		"sz165313",
		"sz165508",
		"sz165509",
		"sz165511",
		"sz165512",
		"sz165513",
		"sz165515",
		"sz165516",
		"sz165517",
		"sz165519",
		"sz165520",
		"sz165521",
		"sz165522",
		"sz165523",
		"sz165524",
		"sz165525",
		"sz165526",
		"sz165528",
		"sz165530",
		"sz165531",
		"sz166001",
		"sz166002",
		"sz166005",
		"sz166006",
		"sz166007",
		"sz166008",
		"sz166009",
		"sz166011",
		"sz166016",
		"sz166023",
		"sz166024",
		"sz166025",
		"sz166027",
		"sz166105",
		"sz166107",
		"sz166109",
		"sz166401",
		"sz166802",
		"sz167001",
		"sz167002",
		"sz167003",
		"sz167301",
		"sz167302",
		"sz167501",
		"sz167503",
		"sz167505",
		"sz167506",
		"sz167508",
		"sz167702",
		"sz168002",
		"sz168101",
		"sz168102",
		"sz168103",
		"sz168104",
		"sz168105",
		"sz168203",
		"sz168204",
		"sz168301",
		"sz168401",
		"sz168701",
		"sz169101",
		"sz169104",
		"sz169105",
		"sz169106",
		"sz169108",
		"sz169201",
		"sh520810",
		"sh520770",
		"sh508050",
		"sh520850",
		"sh520910",
		"sh520610",
		"sh520930",
		"sh530060",
		"sh520760",
		"sh520790",
		"sh520680",
		"sh526000",
		"sh508020",
		"sh526010",
		"sh520730",
		"sh526050",
		"sh508093",
		"sh520710",
		"sh526070",
		"sh508030",
		"sh508603",
		"sh508600",
		"sh508601",
		"sh508602",
		"sh526030",
		"sh520750",
	}
	Start  = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	End    = time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local)
	CsvEnd = time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	Delete930 = true
)

func main() {
	m, err := tdx.NewManage(tdx.WithClients(Clients))
	logs.PanicErr(err)

	if len(Codes) == 0 {
		Codes = m.Codes.GetStockCodes()
	}

	logs.PrintErr(pull(m, Codes))
}

func pull(m *tdx.Manage, codes []string) error {
	b := bar.New(
		bar.WithTotal(int64(len(codes))),
		bar.WithPrefix("xx000000"),
		bar.WithFlush(),
	)
	defer b.Close()
	wg := chans.NewWaitLimit(Goroutine)
	for i, _ := range codes {
		wg.Add()
		go func(code string) {
			defer func() {
				b.Add(1)
				b.Flush()
				wg.Done()
			}()
			b.SetPrefix("[" + code + "]")
			b.Flush()
			var (
				ts  protocol.Trades
				err error
			)
			err = g.Retry(func() error {
				return m.Do(func(c *tdx.Client) error {
					ts, err = pullTrades(c, m.Workday, code)
					return err
				})
			}, Retry, RetryInterval)
			if err != nil {
				b.Logf("[错误] [%s] %s", code, err)
				b.Flush()
				return
			}
			err = save(ts.Klines(), code)
			if err != nil {
				b.Logf("[错误] [%s] %s", code, err)
				b.Flush()
				return
			}
		}(codes[i])
	}
	wg.Wait()
	return nil
}

func pullTrades(c *tdx.Client, w *tdx.Workday, code string) (ts protocol.Trades, err error) {
	resp, err := c.GetKlineMonthAll(code)
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, nil
	}
	start := time.Date(resp.List[0].Time.Year(), resp.List[0].Time.Month(), 1, 0, 0, 0, 0, resp.List[0].Time.Location())
	if Start.After(start) {
		start = Start
	}
	var res *protocol.TradeResp
	w.Range(start, End, func(t time.Time) bool {
		res, err = c.GetHistoryTradeDay(t.Format("20060102"), code)
		if err != nil {
			return false
		}
		ts = append(ts, res.List...)
		return true
	})
	return
}

func save(ks protocol.Klines, code string) error {

	//判断是否需要过滤9.30的数据,ETF和指数没有这个数据
	if Delete930 {
		ks2 := protocol.Klines{}
		for _, v := range ks {
			if v.Time.Hour() == 9 && v.Time.Minute() == 30 {
				continue
			}
			ks2 = append(ks2, v)
		}
		ks = ks2
	}

	//按年分割
	m := map[int]protocol.Klines{}
	for i := range ks {
		if protocol.IsIndex(code) {
			ks[i].Amount = protocol.Price(ks[i].Volume * 100 * 1000)
			ks[i].Volume = 0
		}
		m[ks[i].Time.Year()] = append(m[ks[i].Time.Year()], ks[i])
	}
	for year, ls := range m {

		k1 := toModel(ls)
		k5 := toModel(ls.Merge(5))
		k15 := toModel(ls.Merge(15))
		k30 := toModel(ls.Merge(30))
		k60 := toModel(ls.Merge(60))

		err := insertDB(year, code, k1, k5, k15, k30, k60)
		if err != nil {
			return err
		}
	}
	return exportCsv(ks, code)
}

func toModel(ks protocol.Klines) []any {
	inserts := make([]any, 0, len(ks))
	for _, v := range ks {
		inserts = append(inserts, &KlineBase{
			Date:   v.Time.Unix(),
			Year:   v.Time.Year(),
			Month:  int(v.Time.Month()),
			Day:    v.Time.Day(),
			Hour:   v.Time.Hour(),
			Minute: v.Time.Minute(),
			Open:   v.Open.Float64(),
			High:   v.High.Float64(),
			Low:    v.Low.Float64(),
			Close:  v.Close.Float64(),
			Volume: int(v.Volume),
			Amount: v.Amount.Float64(),
		})
	}
	return inserts
}

func insertDB(year int, code string, k1, k5, k15, k30, k60 []any) error {
	if len(k1) == 0 {
		return nil
	}
	filename := filepath.Join(DatabaseDir, conv.String(year), code+".db")
	db, err := sqlite.NewXorm(filename)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Sync2(new(KlineMinute1), new(KlineMinute5), new(KlineMinute15), new(KlineMinute30), new(KlineMinute60)); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute1), k1); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute5), k5); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute15), k15); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute30), k30); err != nil {
		return err
	}
	if err = _insertDB(db, new(KlineMinute60), k60); err != nil {
		return err
	}
	return nil
}

func _insertDB(db *xorms.Engine, table Timer, inserts []any) error {
	return db.SessionFunc(func(session *xorm.Session) error {
		if _, err := session.Where("ID>0").Delete(table); err != nil {
			return err
		}
		_, err := session.Table(table).Insert(inserts...)
		return err
	})
}

func exportCsv(ks protocol.Klines, code string) error {
	err := _exportCsv(ks, code, "1分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(5), code, "5分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(15), code, "15分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(30), code, "30分钟")
	if err != nil {
		return err
	}
	err = _exportCsv(ks.Merge(60), code, "60分钟")
	if err != nil {
		return err
	}
	return nil
}

func _exportCsv(ks protocol.Klines, code, _type string) error {
	data := [][]any{
		{"日期", "时间", "开盘", "最高", "最低", "收盘", "成交量", "成交额"},
	}
	for _, v := range ks {
		if v.Time.Before(Start) || v.Time.After(End) {
			continue
		}
		if v.Time.After(CsvEnd) {
			continue
		}
		data = append(data, []any{
			v.Time.Format(time.DateOnly),
			v.Time.Format("15:04"),
			v.Open.Float64(),
			v.High.Float64(),
			v.Low.Float64(),
			v.Close.Float64(),
			v.Volume,
			v.Amount.Float64(),
		})
	}
	buf, err := csv.Export(data)
	if err != nil {
		return err
	}
	filename := filepath.Join(CsvDir, _type, code+".csv")
	return oss.New(filename, buf)
}
