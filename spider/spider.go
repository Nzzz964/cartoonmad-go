package spider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/gocolly/colly/queue"
)

var gSpider *Spider = nil

type Spider struct {
	//目标URL
	Target string
	//目标页码 1-20
	Range string
	//Proxy
	Proxy string
	//线程数量
	Thread int
	//请求头
	Headers map[string]string
	//Collector
	Collector *colly.Collector
}

func NewSpider(u string, r string) *Spider {
	gSpider = &Spider{
		Target: u,
		Range:  r,
		Thread: 8,
		Headers: map[string]string{
			"Referer": "https://www.cartoonmad.com",
		},
		Collector: colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"),
		),
	}
	return gSpider
}

func GetSpider() *Spider {
	return gSpider
}

func (s *Spider) SetThread(t int) *Spider {
	s.Thread = t
	return s
}

//设置代理 Set Colly Proxy
func (s *Spider) SetProxy(p string) *Spider {
	s.Proxy = p
	rp, _ := proxy.RoundRobinProxySwitcher(p)
	s.Collector.SetProxyFunc(rp)
	return s
}

//获取Comic对象
func (s *Spider) GetComic() (*Comic, error) {
	c := s.Collector
	cm := Comic{}

	c.OnRequest(func(r *colly.Request) {
		for k, v := range s.Headers {
			r.Headers.Set(k, v)
		}
	})

	c.OnHTML("body > table > tbody > tr:nth-child(1) > td:nth-child(2) > table > tbody > tr:nth-child(3) > td:nth-child(2) > a:nth-child(6)",
		func(e *colly.HTMLElement) {
			b, _ := DecodeBig5([]byte(e.Text))
			cm.Title = b
		},
	)
	//Get all chapter
	c.OnHTML("table:nth-child(3) table td a",
		func(e *colly.HTMLElement) {
			matches := regexp.MustCompile(
				`^/comic/(\d{4})(\d)(\d{3})(\d{4})(\d{3}).html$`,
			).FindStringSubmatch(e.Attr("href"))
			cm.Urls = append(cm.Urls, "https://www.cartoonmad.com/comic/comicpic.asp?file=/"+matches[1]+"/"+matches[3]+"/")
			cm.Total++
		},
	)

	c.OnHTML("table:nth-child(3) table td font",
		func(e *colly.HTMLElement) {
			t, _ := DecodeBig5([]byte(e.Text))
			t = regexp.MustCompile(`\d+`).FindString(t)
			i, _ := strconv.Atoi(t)
			cm.Pages = append(cm.Pages, i)
		},
	)

	if e := c.Visit(s.Target); e != nil {
		fmt.Printf("request: %s error\n", s.Target)
		os.Exit(1)
	}

	rg := strings.Split(s.Range, "-")
	length := len(cm.Urls)
	start, _ := strconv.Atoi(rg[0])
	end, _ := strconv.Atoi(rg[1])
	if 0 < start && start <= length && start <= end && end <= length {
		start--
		cm.Urls = cm.Urls[start:end]
		cm.Pages = cm.Pages[start:end]
		return &cm, nil
	}
	return &cm, fmt.Errorf("目标章节范围不正确 min:%d max:%d", 1, length)
}

func (s *Spider) Start() {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"),
	)
	if s.Proxy != "" {
		rp, _ := proxy.RoundRobinProxySwitcher(s.Proxy)
		c.SetProxyFunc(rp)
	}
	q, _ := queue.New(
		s.Thread,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	comic, err := s.GetComic()

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	title := comic.Title
	chapters := comic.GetChapter()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Referer", "https://www.cartoonmad.cc/")
		r.Headers.Set("Host", "www.cartoonmad.com")
	})

	//ProcessBar
	bar := Bar{}
	//Downloaded Count
	dCount := 0

	c.OnResponse(func(r *colly.Response) {
		url := fmt.Sprint(r.Request.URL)
		matches := regexp.MustCompile(`^https:\/\/www\.cartoonmad\.com\/.*?\/.*?\/(\d*)\/(.*)$`).FindStringSubmatch(url)
		// Download path
		dp := strings.Join([]string{"./downloads/", title, "/第", matches[1], "话/"}, "")
		if _, err := os.Stat(dp); os.IsNotExist(err) {
			os.MkdirAll(dp, 0755)
		}
		r.Save(dp + matches[2])
		dCount++
		bar.Play(int64(dCount))
	})

	c.OnError(func(r *colly.Response, e error) {
		fmt.Printf("\nrequest: %s error retrying...\n", fmt.Sprint(r.Request.URL))
		r.Request.Retry()
	})

	count := 0
	for _, chapter := range chapters {
		for _, url := range chapter.Urls {
			count++
			q.AddURL(url)
		}
	}

	bar.NewOptionWithGraph(0, int64(count), "#")
	q.Run(c)

	fmt.Println()
}
