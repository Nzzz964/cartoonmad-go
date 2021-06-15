package main

import (
	"flag"
	"fmt"

	"madspy/spider"
)

var u = flag.String("u", "", "Require cartoonmad 目标URL")
var r = flag.String("r", "", "Require 目标章节 如: 1-20")
var p = flag.String("p", "", "代理 如: socks5://127.0.0.1:1089")
var t = flag.Int("t", 8, "线程数量 默认:8")

func main() {
	flag.Parse()

	if *u == "" {
		fmt.Println("-u 为必传参数")
		return
	}
	if *r == "" {
		fmt.Println("-r 为必须传参数")
		return
	}

	sp := spider.NewSpider(*u, *r).SetThread(*t)
	if *p != "" {
		sp.SetProxy(*p)
	}
	sp.Start()
}
