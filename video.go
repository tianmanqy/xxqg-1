package main

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func video(ctx context.Context, n int) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(n)*browseLimit)
	defer cancel()

	var url string
	if err := chromedp.Run(ctx, chromedp.AttributeValue(`//a[text()="学习电视台"]`, "href", &url, nil)); err != nil {
		return err
	}

	log.Println("[菜单] 学习电视台", url)

	var album string
	if err := chromedp.Run(
		ctx,
		chromedp.Navigate(url),
		chromedp.AttributeValue(`//div[@class="textWrapper"][text()="学习专题报道"]`, "data-link-target", &album, nil),
	); err != nil {
		return err
	}

	log.Println("[视听] 学习专题报道", album)
	log.Println("计划学习次数:", n)

	var videos []map[string]string
	if err := chromedp.Run(
		ctx,
		chromedp.Navigate(album),
		chromedp.AttributesAll("div.textWrapper", &videos),
	); err != nil {
		return err
	}

	done := listenPclog(ctx)
	for i, v := range videos {
		if i+1 > n {
			break
		}

		start := time.Now()
		url := v["data-link-target"]
		var title string
		if err := chromedp.Run(
			ctx,
			chromedp.Navigate(url),
			chromedp.Text("div.videoSet-article-title", &title),
			chromedp.ActionFunc(func(context.Context) error {
				log.Printf("#视频%d %s %s", i+1, title, url)
				return nil
			}),
			chromedp.WaitVisible("div.outter"),
			chromedp.Sleep(time.Second),
			chromedp.Click("div.outter"),
		); err != nil {
			return err
		}

		select {
		case <-time.After(browseLimit):
			log.Print("单个视频超时！")
		case <-done:
		}

		log.Printf("时长：%s", time.Since(start))
	}

	log.Print("视听学习完毕！")
	return nil
}
