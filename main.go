package main

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	switch os.Args[1] {
	case "collect":
		collectPC()
	case "stat":
		stat() //TODO
	default:
		pt("no such command %s\n", os.Args[1])
	}
}

var urlPatterns = map[string]string{
	"梦丹铃": "http://search.jd.com/s_new.php?keyword=%E6%A2%A6%E4%B8%B9%E9%93%83&enc=utf-8&qrst=1&rt=1&stop=1&vt=2&sttr=1&psort=5&page=2&s=31&scrolling=y&pos=30&log_id=1462592650.11280&tpl=3_L&show_items=10330535290,10330493682,10330450765,10330360082,10330332783,10330311814,10330199310,10326228667,10326016628,10325966377,10325804449,10325751033,10325670583,10324065938,10323953038,10323903801,10323820315,10323819817,10323632332,10323622762,10323567806,10323548065,10323493250,10318659930,10314739525,10314721519,10314713506,10314696274,10314678889,10314675153",
	"穆美":  "http://search.jd.com/s_new.php?keyword=%E7%A9%86%E7%BE%8E&enc=utf-8&qrst=1&rt=1&stop=1&vt=2&sttr=1&bs=1&psort=5&ev=exbrand_%E7%A9%86%E7%BE%8E%EF%BC%88MUMERI%EF%BC%89%40&page=2&s=31&scrolling=y&pos=30&log_id=1462592611.21960&tpl=3_L&show_items=10298359607,10298309476,10295834215,10280729475,10279762931,10279527201,10276160772,10276137936,10275098970,10274670684,10270425322,10265570852,10265548760,10265529913,10263034136,10262998539,10262920827,10262873344,10262376868,10262134863,10262104486,10261983123,10261915340,10261857219,10261857121,10261835435,10261814711,10261777661,10261752097,10261738643",
	"塞巴莉": "http://search.jd.com/s_new.php?keyword=%E5%A1%9E%E5%B7%B4%E8%8E%89&enc=utf-8&qrst=1&rt=1&stop=1&vt=2&sttr=1&psort=5&page=2&s=31&scrolling=y&pos=30&log_id=1462596985.85539&tpl=3_L&show_items=10329948685,10329843299,10327810795,10327787716,10327582491,10326260215,10326147726,10325958744,10325885259,10316117832,10316047874,10316021879,10315945247,10315846867,10315831937,10313836601,10313766368,10313678135,10311259229,10311077025,10310958273,10299565424,10299486497,10299057945,10299015490,10298956495,10298491292,10298457782,10298422776,10298389823",
	"麦拉迪": "http://search.jd.com/s_new.php?keyword=%E9%BA%A6%E6%8B%89%E8%BF%AA&enc=utf-8&qrst=1&rt=1&stop=1&vt=2&sttr=1&bs=1&psort=5&ev=exbrand_%E9%BA%A6%E6%8B%89%E8%BF%AA%EF%BC%88MCRALDE%EF%BC%89%40&page=2&s=31&scrolling=y&pos=30&log_id=1462599901.95488&tpl=3_L&show_items=10328485891,10328472125,10328415479,10326682473,10320807810,10320778660,10320731476,10320716558,10319349673,10319246505,10318986480,10318938709,10317458584,10317364469,10314395568,10314351397,10312082881,10296922525,10289659502,10283529368,10283378176,10283254538,10282796438,10280591830,10280294156,10280108211,10280046727,10280023090,10280005430,10278231207",
}

func collectPC() {
	date := time.Now().Format("2006-01-02")

	for _, psort := range []string{"2", "3", "4", "5"} {
		pt("----- psort %s -----\n", psort)
		for shopName, pattern := range urlPatterns {
			// parse
			u, err := url.Parse(pattern)
			ce(err, "parse url")
			values := u.Query()

			// adjust
			values["scrolling"] = []string{"y"}
			values["psort"] = []string{psort}
			delete(values, "show_items")
			delete(values, "log_id")
			delete(values, "pos")
			delete(values, "s")

			// shop id
			var shopIds []int
		get_shop_id:
			err = db.Select(&shopIds, `SELECT shop_id FROM shops 
			WHERE name = $1`,
				shopName)
			ce(err, "get shop id")
			var shopId int
			if len(shopIds) == 0 {
				db.MustExec(`INSERT INTO shops (name) VALUES ($1)`, shopName)
				goto get_shop_id
			} else {
				shopId = shopIds[0]
			}
			pt("shop %d\n", shopId)

			spus := make(map[string]bool)

			// pages
			for page := 1; ; page++ {
				pt("page %d ", page)
				os.Stdout.Sync()
				values["page"] = []string{strconv.Itoa(page)}
				pageUrl := "http://search.jd.com/s_new.php?" + values.Encode()

				doc, err := getDoc(pageUrl)
				ce(err, "get doc %s", pageUrl)
				lis := doc.Find("body > li")
				if lis.Length() == 0 {
					break
				}
				tx := db.MustBegin()
				lis.Each(func(i int, se *goquery.Selection) {
					spu, _ := se.Attr("data-spu")
					if len(spu) == 0 {
						panic(me(nil, "no spu. url %s", pageUrl))
					}
					spus[spu] = true
					imgUrl, _ := se.Find("div.p-img img").Attr("src")
					if len(imgUrl) == 0 {
						imgUrl, _ = se.Find("div.p-img img").Attr("data-lazy-img")
					}
					if len(imgUrl) == 0 {
						panic(me(nil, "no image. url %s", pageUrl))
					}
					if !strings.HasPrefix(imgUrl, "http") {
						imgUrl = "http:" + imgUrl
					}
					price, _ := se.Find("div.p-price strong").Attr("data-price")
					if len(price) == 0 {
						panic(me(nil, "no price. url %s", pageUrl))
					}
					title, _ := se.Find("div.p-name a").Attr("title")
					if len(title) == 0 {
						panic(me(nil, "no title. url %s", pageUrl))
					}
					comments := se.Find("div.p-commit a").Text()
					if len(comments) == 0 {
						panic(me(nil, "no comments. url %s", pageUrl))
					}

					// insert
					tx.MustExec(`INSERT INTO items
					(spu, shop_id, added_date)
					VALUES ($1, $2, $3)
					ON CONFLICT (spu) DO NOTHING`,
						spu,
						shopId,
						date,
					)
					tx.MustExec(`INSERT INTO infos
					(spu, date, price, comments, title, image_url)
					VALUES ($1, $2, $3, $4, $5, $6)
					ON CONFLICT (spu, date) DO UPDATE SET
					price = $3, comments = $4, title = $5, image_url = $6
					`,
						spu,
						date,
						price,
						comments,
						title,
						imgUrl,
					)
				})

				// commit
				ce(tx.Commit(), "commit")

				// sleep
				//time.Sleep(time.Millisecond * 500)
			}
			pt("\n")

			pt("collect %d spus\n", len(spus))
		}
	}
}

func getDoc(pageUrl string) (doc *goquery.Document, err error) {
	defer ct(&err)
	retry := 10
do:
	resp, err := http.Get(pageUrl)
	if err != nil {
		if retry > 0 {
			retry--
			goto do
		}
		ce(err, "get page %s", pageUrl)
	}
	defer resp.Body.Close()
	doc, err = goquery.NewDocumentFromResponse(resp)
	if err != nil {
		if retry > 0 {
			retry--
			goto do
		}
		ce(err, "new doc from resp %s", pageUrl)
	}
	return
}
