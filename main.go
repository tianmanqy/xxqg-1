package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	homeURL     = "https://www.xuexi.cn/"
	pointsURL   = "https://pc.xuexi.cn/points/my-points.html"
	pointsAPI   = "https://pc-proxy-api.xuexi.cn/api/score/days/listScoreProgress"
	loginURL    = "https://pc.xuexi.cn/points/login.html"
	practiceURL = "https://pc.xuexi.cn/points/exam-practice.html"
	weeklyURL   = "https://pc.xuexi.cn/points/exam-weekly-list.html"
	paperURL    = "https://pc.xuexi.cn/points/exam-paper-list.html"
	pclogURL    = "https://iflow-api.xuexi.cn/logflow/api/v1/pclog"
)

const (
	loginLimit  = 2 * time.Minute
	pointsLimit = 10 * time.Second
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
	articleCount = 12
	videoCount   = 12
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

	res := getPoints(ctx)
	log.Print(res)
	t := res.CreateTask()
	if reflect.DeepEqual(t, task{}) {
		log.Print("学习积分已达上限！")
		return
	}

	start := time.Now()

	dividingLine()
	for t.practice {
		checkError("每日答题", exam(ctx, practiceURL, "", practiceCount, practiceLimit))
		dividingLine()
		t = getPoints(ctx).CreateTask()
	}
	if t.weekly {
		checkError("每周答题", exam(ctx, weeklyURL, weeklyClass, weeklyCount, weeklyLimit))
		dividingLine()
	}
	if t.paper {
		checkError("专项答题", exam(ctx, paperURL, paperClass, paperCount, paperLimit))
		dividingLine()
	}
	if t.article > 0 {
		checkError("选读文章", article(ctx, t.article))
		dividingLine()
	}
	if t.video > 0 {
		checkError("视听学习", video(ctx, t.video))
		dividingLine()
	}

	log.Printf("学习完成！总耗时：%s", time.Since(start))

	time.Sleep(time.Second)
	log.Print(getPoints(ctx))
}

func checkError(task string, err error) {
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Printf("%s: 任务超时或没有可用资源", task)
		} else {
			log.Printf("%s: %s", task, err)
		}
	}
}

func dividingLine() {
	io.WriteString(log.Default().Writer(), strings.Repeat("=", 100)+"\r\n")
}
