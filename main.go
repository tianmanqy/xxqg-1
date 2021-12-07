package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	homeURL     = "https://www.xuexi.cn/"
	loginURL    = "https://pc.xuexi.cn/points/login.html"
	practiceURL = "https://pc.xuexi.cn/points/exam-practice.html"
	weeklyURL   = "https://pc.xuexi.cn/points/exam-weekly-list.html"
	paperURL    = "https://pc.xuexi.cn/points/exam-paper-list.html"
	pclogURL    = "https://iflow-api.xuexi.cn/logflow/api/v1/pclog"
)

const (
	loginLimit  = time.Minute
	examLimit   = 15 * time.Second
	browseLimit = 45 * time.Second
)

const (
	practiceCount = 5
	practiceLimit = practiceCount * examLimit
)

const (
	weeklyClass = "week"
	weeklyCount = 5
	weeklyLimit = weeklyCount * examLimit
)

const (
	paperClass = "item"
	paperCount = 10
	paperLimit = paperCount * examLimit
)

const (
	articalCount = 12
	articalLimit = articalCount * browseLimit
)

const (
	videoCount = 12
	videoLimit = videoCount * browseLimit
)

func main() {
	defer func() {
		fmt.Println("Press enter key to exit . . .")
		fmt.Scanln()
	}()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	if err := listenFetch(ctx); err != nil {
		log.Print(err)
		return
	}

	loginCtx, cancel := context.WithTimeout(ctx, loginLimit)
	log.Print("请先扫码登录")
	if err := chromedp.Run(
		loginCtx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible("span.refresh"),
		chromedp.EvaluateAsDevTools(`$("span.refresh").scrollIntoViewIfNeeded()`, nil),
		chromedp.WaitVisible("span.logged-text"),
	); err != nil {
		log.Print(err)
		return
	}
	log.Print("登录成功")
	cancel()

	start := time.Now()

	if err := exam(ctx, practiceURL, "", practiceCount, practiceLimit); err != nil {
		log.Print(err)
	}
	if err := exam(ctx, weeklyURL, weeklyClass, weeklyCount, weeklyLimit); err != nil {
		log.Print(err)
	}
	if err := exam(ctx, paperURL, paperClass, paperCount, paperLimit); err != nil {
		log.Print(err)
	}
	if err := artical(ctx); err != nil {
		log.Print(err)
	}
	if err := video(ctx); err != nil {
		log.Print(err)
	}

	log.Printf("学习完成！总耗时：%s", time.Since(start))
}
