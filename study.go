package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

const (
	studyLimit = 31 * time.Second
	modeLimit  = 5 * time.Second
)

var (
	articles = make(chan item, articleNumber)
	videos   = make(chan item, videoNumber)
)

var (
	errTitleMismatch = errors.New("标题不匹配")
	errModeMismatch  = errors.New("模式不匹配")
)

func study(ctx context.Context, article, video int64, status *status) {
	if article > 0 {
		log.Println("计划阅读文章次数:", article)
	}
	if video > 0 {
		log.Println("计划浏览视频次数:", video)
	}

	maxRetry := 5
	var n1, n2 int64
	for items := range locals {
		if n1 < article {
			for i := range items.articles {
				log.Println("#文章", n1+1)
				if err := studySingle(ctx, i); err != nil {
					log.Println(i, err)
					if maxRetry > 0 {
						maxRetry--
						status.addArticle(1)
						continue
					}
				}
				if n1++; n1 == article {
					if n2 == video {
						if !allGet {
							locals <- items
						}
						return
					}
					break
				}
			}
		}
		if n2 < video {
			for i := range items.videos {
				log.Println("#视频", n2+1)
				if err := studySingle(ctx, i); err != nil {
					log.Println(i, err)
					if maxRetry > 0 {
						maxRetry--
						status.addVideo(1)
						continue
					}
				}
				if n2++; n2 == video {
					if n1 == article {
						if !allGet {
							locals <- items
						}
						return
					}
					break
				}
			}
		}
		if !allGet && (len(items.articles) > 0 || len(items.videos) > 0) {
			locals <- items
		}
	}

	if n1 < article {
		for i := range articles {
			log.Println("#文章", n1+1)
			if err := studySingle(ctx, i); err != nil {
				log.Println(i, err)
				if maxRetry > 0 {
					maxRetry--
					status.addArticle(1)
					continue
				}
			}
			if n1++; n1 == article {
				break
			}
		}
	}
	if n2 < video {
		for i := range videos {
			log.Println("#视频", n2+1)
			if err := studySingle(ctx, i); err != nil {
				log.Println(i, err)
				if maxRetry > 0 {
					maxRetry--
					status.addVideo(1)
					continue
				}
			}
			if n2++; n2 == video {
				break
			}
		}
	}
}

var last string

func studySingle(ctx context.Context, item item) error {
	prepare, cancel := context.WithTimeout(ctx, studyLimit)
	defer cancel()

	if item.url != last {
		if err := chromedp.Run(prepare, chromedp.Navigate(item.url), chromedp.WaitVisible("div.list_text")); err != nil {
			return err
		}
		last = item.url
	}

	var sel any
	for _, i := range []any{
		item.byIndex(),
		item.byTitle(),
	} {
		ctx, cancel := context.WithTimeout(prepare, time.Second)
		defer cancel()

		var title string
		if err := chromedp.Run(ctx, chromedp.Text(i, &title)); err != nil {
			log.Println(title, i, err)
			continue
		}
		if item.compare(title) {
			sel = i
			break
		}
	}
	if sel == nil {
		return errTitleMismatch
	}

	log.Println(item.date, item.title)

	if err := chromedp.Run(prepare, chromedp.Click(sel)); err != nil {
		return err
	}
	time.Sleep(time.Second)

	targets, err := chromedp.Targets(ctx)
	if err != nil {
		return err
	}

	var study context.Context
	for _, t := range targets {
		if t.Type == "page" {
			if strings.Contains(t.URL, "special.html") {
				return errModeMismatch
			} else if strings.Contains(t.URL, "detail.html") || t.URL == "about:blank" {
				ctx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(t.TargetID))
				defer cancel()

				study, cancel = context.WithTimeout(ctx, studyLimit)
				defer cancel()

				if err := chromedp.Run(study, chromedp.WaitVisible("div.content")); err != nil {
					return err
				}

				modeCtx, cancel := context.WithTimeout(study, modeLimit)
				defer cancel()

				var nodes []*cdp.Node
				var sel any
				switch item.Type {
				case articleItem:
					sel = "div.render-detail-title"
				case videoItem:
					sel = "div.outter"
				}
				if err := chromedp.Run(modeCtx, chromedp.Nodes(sel, &nodes)); err != nil {
					return errModeMismatch
				}

				if err := chromedp.Run(study, chromedp.MouseClickNode(nodes[0])); err != nil {
					return err
				}
			}
		}
	}

	if study == nil {
		return errors.New("page not found")
	} else {
		<-study.Done()
		if err := study.Err(); err != nil && err != context.DeadlineExceeded {
			return err
		}
	}

	return nil
}
