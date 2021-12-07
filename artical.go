package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

func artical(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, articalLimit)
	defer cancel()

	if err := chromedp.Run(
		ctx,
		chromedp.Navigate(homeURL),
		chromedp.Click(`div[data-data-id="important-news-title"] div.extra>span`, chromedp.NodeVisible),
	); err != nil {
		return err
	}

	time.Sleep(time.Second)
	targets, err := chromedp.Targets(ctx)
	if err != nil {
		return err
	}

	var target *target.Info
	for _, i := range targets {
		if i.Type == "page" && i.URL != homeURL {
			target = i
			break
		}
	}
	if target == nil {
		return fmt.Errorf("no target found")
	}
	log.Println("[阅读] 重要新闻", target.URL)

	navCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
	defer cancel()

	var nodes []*cdp.Node
	if err := chromedp.Run(
		navCtx,
		chromedp.WaitVisible("div.text-link-item-title"),
		chromedp.Nodes("div.text-link-item-title div.text-wrap>span", &nodes),
	); err != nil {
		return err
	}

	for i, node := range nodes {
		if i+1 > articalCount {
			break
		}

		start := time.Now()
		if err := chromedp.Run(
			navCtx,
			chromedp.MouseClickNode(node),
		); err != nil {
			return err
		}

		var done <-chan struct{}
		var cancel context.CancelFunc
		targets, err := chromedp.Targets(ctx)
		if err != nil {
			return err
		}
		for _, t := range targets {
			if t.Type == "page" && t.URL != homeURL && t.URL != target.URL {
				var articalCtx context.Context
				articalCtx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(t.TargetID))

				done = listenPclog(articalCtx)
				time.Sleep(time.Second)
				if err := chromedp.Run(
					articalCtx,
					chromedp.Click("span.link-text", chromedp.NodeVisible),
				); err != nil {
					cancel()
					return err
				}

				var title string
				if len(node.Children) != 0 {
					title = node.Children[0].NodeValue
				}

				log.Printf("#文章%d %s %s", i+1, title, t.URL)
				break
			}
		}

		select {
		case <-time.After(browseLimit):
			log.Print("单篇文章超时！")
		case <-done:
		}
		if cancel != nil {
			cancel()
		}

		log.Printf("时长：%s", time.Since(start))
	}

	log.Print("选读文章完毕！")
	return nil
}
