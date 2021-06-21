package spider

import (
	"fmt"
	"strings"
)

type Comic struct {
	//名字
	Title string
	//总章节数
	Total int
	//所有章节Url
	Urls []string
	//Pages
	Pages []int
}

func (c *Comic) GetChapter() []Chapter {
	size := len(c.Urls)
	chapters := make([]Chapter, size)

	for i := 0; i < size; i++ {
		count := c.Pages[i]
		for k := 1; k <= count; k++ {
			s := fmt.Sprint(k)
			s = "000"[len(s):] + s
			chapters[i].Urls = append(
				chapters[i].Urls,
				strings.Join([]string{c.Urls[i], s, "&rimg=1"}, ""),
			)
		}
	}

	return chapters
}
