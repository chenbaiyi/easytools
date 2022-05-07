package music

import (
	"github.com/astaxie/beego"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

func StartSpider() {
	// 实例化默认收集器
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.82 Safari/537.36"),
		colly.AllowURLRevisit(),
		)

	// 仅访问域
	c.AllowedDomains = []string{"music.163.com"}

	// 表示抓取时异步的
	//c.Async = true

	// 限制采集规则
	/*
		在Colly里面非常方便控制并发度，只抓取符合某个(些)规则的URLS
		colly.LimitRule{DomainGlob: "*.douban.*", Parallelism: 5}，表示限制只抓取域名是douban(域名后缀和二级域名不限制)的地址，当然还支持正则匹配某些符合的 URLS

		Limit方法中也限制了并发是5。为什么要控制并发度呢？因为抓取的瓶颈往往来自对方网站的抓取频率的限制，如果在一段时间内达到某个抓取频率很容易被封，所以我们要控制抓取的频率。
		另外为了不给对方网站带来额外的压力和资源消耗，也应该控制你的抓取机制。
	*/
	// err := c.session.Limit(&colly.LimitRule{DomainGlob: "*.quotes.*", Parallelism: 5})
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// 访问地址
	url := "https://music.163.com/discover/toplist"

	extensions.RandomUserAgent(c)

	c.OnHTML(".pager .next a", func(e *colly.HTMLElement) {
	// 获取属性值
	link := e.Attr("href")
		beego.Info("Link found: %q -> %s\n", e.Text, link)

	})

	c.OnResponse(func(r *colly.Response) {
		beego.Info(string(r.Body))
	})

	c.OnError(func(_ *colly.Response, err error) {
		beego.Info("Something went wrong:", err)
	})

	// 结束
	c.OnScraped(func(r *colly.Response) {
		beego.Info("Finished", r.Request.URL)
	})

	// 开始爬取 url
	err2 := c.Visit(url)
	if err2 != nil {
		beego.Info("err2", err2)
	}

	// 采集等待结束
	c.Wait()

}