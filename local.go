package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
)

var localList = []local{
	"bj", "tj", "he", "sx", "nmg",
	"ln", "jl", "hlj", "sh", "js",
	"zj", "ah", "fj", "jx", "sd",
	"ha", "hb", "hn", "gd", "gx",
	"hq", "cq", "sc", "gz", "yn",
	"tibet", "sn", "gs", "qh", "nx",
	"xj", "xjbt", "zyqy", "zgkxy",
}

type local string

func (s local) url() string {
	return fmt.Sprintf("https://%s.xuexi.cn/", s)
}

func (s local) json() *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`local/%s/channel/.+\.json`, s))
}

func (s local) items(ctx context.Context, status *status, date time.Time) (items items, err error) {
	if err = chromedp.Run(ctx, chromedp.Navigate(s.url()), chromedp.Click(`//a[text()="查看更多"]`)); err != nil {
		return
	}

	var targets []*target.Info
	var target *target.Info
	for loop := 5; loop > 0; loop-- {
		time.Sleep(time.Second * 3)

		targets, err = chromedp.Targets(ctx)
		if err != nil {
			return
		}

		for _, i := range targets {
			if i.Type == "page" && strings.Contains(i.URL, "list.html") {
				target = i
				loop = 0
				break
			}
		}
	}
	if target == nil {
		targets, _ := chromedp.Targets(ctx)
		for _, i := range targets {
			if i.Type == "page" && i.URL != s.url() {
				_, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(i.TargetID))
				cancel()
			}
		}
		err = fmt.Errorf("%s无法打开页面", s)
		return
	}

	ctx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
	defer cancel()

	done := chrome.ListenEvent(ctx, s.json(), "GET", true)
	go chromedp.Run(ctx, chromedp.Reload())
	select {
	case <-ctx.Done():
		err = fmt.Errorf("无法获取数据: %s", ctx.Err())
	case event := <-done:
		var res []item
		res, err = parseItems(s, target.URL, event)
		if err != nil {
			return
		}

		items.articles, items.videos = make(chan item, articleNumber), make(chan item, videoNumber)
		for _, i := range res {
			if i.date != "" {
				var c chan item
				switch i.date {
				case date.Format("2006-01-02"):
					switch i.Type {
					case articleItem:
						c = items.articles
						status.reduceArticle()
					case videoItem:
						c = items.videos
						status.reduceVideo()
					}
				case date.AddDate(0, 0, -1).Format("2006-01-02"):
					switch i.Type {
					case articleItem:
						c = articles
					case videoItem:
						c = videos
					}
				}
				if len(c) < cap(c) {
					c <- i
				}
			}
		}
		close(items.articles)
		close(items.videos)
		if status.article.Load() <= 0 && status.video.Load() <= 0 {
			status.Pause()
		}
	}
	return
}
