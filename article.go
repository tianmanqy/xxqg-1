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

func article(ctx context.Context, n int) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(n)*browseLimit)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Navigate(homeURL)); err != nil {
		return err
	}

	var target *target.Info
	done := make(chan struct{}, 1)
	go func() {
		for {
			chromedp.Run(ctx, chromedp.Click(`//span[text()="重要新闻"]`, chromedp.NodeVisible))
			time.Sleep(time.Second)
			targets, _ := chromedp.Targets(ctx)
			for _, i := range targets {
				if i.Type == "page" && i.URL != homeURL {
					target = i
					close(done)
					return
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("no target found")
	case <-done:
	}

	log.Println("[阅读] 重要新闻", target.URL)
	log.Println("计划学习次数:", n)

	navCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
	defer cancel()

	var nodes []*cdp.Node
	if err := chromedp.Run(navCtx, chromedp.Nodes("div.text-wrap>span", &nodes, chromedp.NodeVisible)); err != nil {
		return err
	}

	for i, node := range nodes {
		if i+1 > n {
			break
		}

		start := time.Now()
		if err := chromedp.Run(navCtx, chromedp.MouseClickNode(node)); err != nil {
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
				var articleCtx context.Context
				articleCtx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(t.TargetID))

				done = listenPclog(articleCtx)
				time.Sleep(time.Second)
				if err := chromedp.Run(articleCtx, chromedp.Click("span.link-text", chromedp.NodeVisible)); err != nil {
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
