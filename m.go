package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var keywords = []string{
	"穆美",
	"塞巴莉",
	"梦丹铃",
}

type Info struct {
	Cid1       string // 一级类目
	Cid2       string // 二级类目
	Catid      string // 三级类目
	Good       string // 好评率
	JdPrice    string // 价格
	WareId     string // sku
	TotalCount string // 评价量
	Wname      string // 标题
	LongImgUrl string // 图
}

// 不能工作
func collectMobile() {
	date := time.Now().Format("2006-01-02")

	for _, keyword := range keywords {
		var shopIds []int
	get_shop_id:
		err := db.Select(&shopIds, `SELECT shop_id FROM shops 
			WHERE name = $1`,
			keyword)
		ce(err, "get shop id")
		var shopId int
		if len(shopIds) == 0 {
			db.MustExec(`INSERT INTO shops (name) VALUES ($1)`, keyword)
			goto get_shop_id
		} else {
			shopId = shopIds[0]
		}
		pt("shop %d %s\n", shopId, keyword)

		skus := make(map[string]bool)
		var skusLock sync.Mutex
		processInfos := func(infos []Info) {
			tx := db.MustBegin()
			for _, info := range infos {
				skusLock.Lock()
				skus[info.WareId] = true
				skusLock.Unlock()
				tx.MustExec(`INSERT INTO items 
					(sku, shop_id, category, added_date)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (sku) DO NOTHING`,
					info.WareId,
					shopId,
					info.Cid1+","+info.Cid2+","+info.Catid,
					date,
				)
				price, err := strconv.ParseFloat(info.JdPrice, 64)
				ce(err, "parse price %s", info.JdPrice)
				info.Good = strings.Replace(info.Good, "%", "", -1)
				info.Good = strings.Replace(info.Good, "暂无评价", "0", -1)
				if len(info.TotalCount) == 0 {
					info.TotalCount = "0"
				}
				tx.MustExec(`INSERT INTO infos 
					(sku, date, good_rate, price, comments, title, image_url)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (sku, date) DO UPDATE 
					SET good_rate = $3, price = $4, comments = $5, 
						title = $6, image_url = $7`,
					info.WareId,
					date,
					info.Good,
					price,
					info.TotalCount,
					info.Wname,
					info.LongImgUrl,
				)
			}
			ce(tx.Commit(), "commit")
		}

		infos, total, err := GetKeywordPage(keyword, 1)
		ce(err, "get infos")
		processInfos(infos)
		maxPage := (total / 10) + 1

		wg := new(sync.WaitGroup)
		wg.Add(maxPage - 1)
		sem := make(chan bool, 8)
		for page := 2; page <= maxPage; page++ {
			page := page
			sem <- true
			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()
				pt("page %d / %d\n", page, maxPage)
				infos, _, err = GetKeywordPage(keyword, page)
				ce(err, "get infos")
				processInfos(infos)
			}()
		}
		wg.Wait()

		pt("collected %d items\n", len(skus))
	}

}

func GetKeywordPage(keyword string, page int) (infos []Info, total int, err error) {
	defer ct(&err)
	reqUrl := "http://so.m.jd.com/ware/searchList.action"
	form := url.Values{
		"_format_": {"json"},
		"page":     {strconv.Itoa(page)},
		"keyword":  {keyword},
		"":         {""},
	}
	req, err := http.NewRequest("POST", reqUrl, strings.NewReader(form.Encode()))
	ce(err, "new request")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://so.m.jd.com")
	//req.Header.Set("Referer", "http://so.m.jd.com/ware/search.action?sid=39661583a3d28d872d9fe529d611eadd&keyword=%E7%A9%86%E7%BE%8E&catelogyList=")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	retry := 10
do:
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if retry > 0 {
			retry--
			goto do
		}
		ce(err, "get page %s %d", keyword, page)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if retry > 0 {
			retry--
			goto do
		}
		ce(err, "get content")
	}

	var wrap struct {
		AreaName string
		Value    string
	}
	err = json.Unmarshal(content, &wrap)
	ce(err, "unmarshal wrap")
	pt("\n%s\n\n", wrap.Value)

	var data struct {
		WareCount int
		WareList  []Info
	}
	err = json.Unmarshal([]byte(wrap.Value), &data)
	ce(err, "unmarshal data")

	return data.WareList, data.WareCount, err
}
