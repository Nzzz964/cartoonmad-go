package spider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/gocolly/colly/queue"
)

var ua = colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36")

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

func collyFactory() *colly.Collector {
	return colly.NewCollector(
		ua,
	)
}

func NewSpider(u string, r string) *Spider {
	return &Spider{
		Target: u,
		Range:  r,
		Thread: 8,
		Headers: map[string]string{
			"Referer": "https://www.cartoonmad.com",
		},
		Collector: collyFactory(),
	}
}

func (s *Spider) SetThread(t int) *Spider {
	s.Thread = t
	return s
}

//设置代理 Set Colly Proxy
func (s *Spider) SetProxy(p string) *Spider {
	s.Proxy = p
	proxyFunc, _ := proxy.RoundRobinProxySwitcher(p)
	s.Collector.SetProxyFunc(proxyFunc)
	return s
}

//获取Comic对象
func (s *Spider) GetComic() (*Comic, error) {
	collector := s.Collector
	comic := Comic{}

	collector.OnRequest(func(r *colly.Request) {
		for k, v := range s.Headers {
			r.Headers.Set(k, v)
		}
	})

	//获取 Title
	collector.OnHTML("td:nth-child(2) > a:nth-child(6)",
		func(e *colly.HTMLElement) {
			b, _ := DecodeBig5([]byte(e.Text))
			comic.Title = b
		},
	)
	//获取所有章节
	collector.OnHTML("table:nth-child(3) table td a",
		func(e *colly.HTMLElement) {
			matches := regexp.MustCompile(
				`^/comic/(\d{4})(\d)(\d{3})(\d{4})(\d{3}).html$`,
			).FindStringSubmatch(e.Attr("href"))
			comic.Urls = append(comic.Urls, "https://www.cartoonmad.com/comic/comicpic.asp?file=/"+matches[1]+"/"+matches[3]+"/")
			comic.Total++
		},
	)
	//获取页数
	collector.OnHTML("table:nth-child(3) table td font",
		func(e *colly.HTMLElement) {
			text, _ := DecodeBig5([]byte(e.Text))
			match := regexp.MustCompile(`\d+`).FindString(text)
			page, _ := strconv.Atoi(match)
			comic.Pages = append(comic.Pages, page)
		},
	)

	if err := collector.Visit(s.Target); err != nil {
		fmt.Printf("request: %s error\n", s.Target)
		os.Exit(1)
	}

	rg := strings.Split(s.Range, "-")
	length := len(comic.Urls)
	start, _ := strconv.Atoi(rg[0])
	end, _ := strconv.Atoi(rg[1])

	if 0 < start && start <= length && start <= end && end <= length {
		start--
		comic.Urls = comic.Urls[start:end]
		comic.Pages = comic.Pages[start:end]
		return &comic, nil
	}

	return &comic, fmt.Errorf("目标章节范围不正确 min:%d max:%d", 1, length)
}

func (s *Spider) Start() {
	collector := collyFactory()

	if s.Proxy != "" {
		proxyFunc, _ := proxy.RoundRobinProxySwitcher(s.Proxy)
		collector.SetProxyFunc(proxyFunc)
	}

	queue, _ := queue.New(
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

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Referer", "https://www.cartoonmad.cc/")
		r.Headers.Set("Host", "www.cartoonmad.com")
	})

	bar := Bar{}
	downloads := 0
	var mutex sync.Mutex

	collector.OnResponse(func(r *colly.Response) {
		url := fmt.Sprint(r.Request.URL)
		matches := regexp.MustCompile(`^https:\/\/.*?\/.*?\/.*?\/(\d*)\/(.*)$`).FindStringSubmatch(url)

		osPath := strings.Join([]string{"./downloads/", title, "/第", matches[1], "话/"}, "")
		if _, err := os.Stat(osPath); os.IsNotExist(err) {
			os.MkdirAll(osPath, 0755)
		}
		r.Save(osPath + matches[2])

		mutex.Lock()
		downloads++
		bar.Play(int64(downloads))
		mutex.Unlock()
	})

	collector.OnError(func(r *colly.Response, e error) {
		fmt.Printf("\nrequest: %s error retrying...\n", fmt.Sprint(r.Request.URL))
		r.Request.Retry()
	})

	count := 0
	for _, chapter := range chapters {
		for _, url := range chapter.Urls {
			count++
			queue.AddURL(url)
		}
	}

	bar.NewOptionWithGraph(0, int64(count), "#")
	queue.Run(collector)

	fmt.Println()
}
