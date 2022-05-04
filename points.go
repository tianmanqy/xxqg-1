package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type task struct {
	practice, weekly, paper bool
	article, video          int
}

type pointsResult struct {
	Data struct {
		TotalScore   int
		TaskProgress []struct {
			Title        string
			CurrentScore int
			DayMaxScore  int
		}
	}
}

func getPoints(ctx context.Context) (res pointsResult) {
	ctx, cancel := context.WithTimeout(ctx, pointsLimit)
	defer cancel()

	var id network.RequestID
	done := make(chan struct{}, 1)
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if strings.HasPrefix(ev.Request.URL, pointsAPI) {
				id = ev.RequestID
			}
		case *network.EventLoadingFinished:
			if ev.RequestID == id {
				close(done)
			}
		}
	})

	if err := chromedp.Run(ctx, chromedp.Navigate(pointsURL)); err != nil {
		log.Print(err)
		return
	}

	select {
	case <-ctx.Done():
		log.Print(ctx.Err())
		return
	case <-done:
	}

	var b []byte
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			b, err = network.GetResponseBody(id).Do(ctx)
			return
		}),
	); err != nil {
		log.Print(err)
		return
	}

	if err := json.Unmarshal(b, &res); err != nil {
		log.Print(err)
	}

	return
}

func (res pointsResult) CreateTask() task {
	t := task{true, true, true, articleCount, videoCount}
	for _, i := range res.Data.TaskProgress {
		switch i.Title {
		case "每日答题":
			if i.CurrentScore == 5 {
				t.practice = false
			}
		case "每周答题":
			if i.CurrentScore > 1 {
				t.weekly = false
			}
		case "专项答题":
			if i.CurrentScore > 1 {
				t.paper = false
			}
		case "我要选读文章":
			if i.CurrentScore < 6 {
				t.article = articleCount
			} else {
				t.article = articleCount - (i.CurrentScore-6)*2
			}
		case "视听学习时长":
			t.video = videoCount - i.CurrentScore*2
		}
	}
	return t
}

func (res pointsResult) String() string {
	m := make(map[string]string)
	for _, i := range res.Data.TaskProgress {
		m[i.Title] = fmt.Sprintf("%d分/%d分", i.CurrentScore, i.DayMaxScore)
	}

	output := fmt.Sprintf("当前积分: %d\n", res.Data.TotalScore)
	for _, i := range []string{"登录", "每日答题", "每周答题", "专项答题", "我要选读文章", "视听学习", "视听学习时长"} {
		output += fmt.Sprintf("%s: %s\n", i, m[i])
	}

	return output
}
