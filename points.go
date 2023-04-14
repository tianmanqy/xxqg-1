package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
)

const (
	pointsURL = "https://pc.xuexi.cn/points/my-points.html"
	pointsAPI = "https://pc-proxy-api.xuexi.cn/delegate/score/days/listScoreProgress"

	pointsLimit = 15 * time.Second

	articleNumber = 12
	videoNumber   = 12
)

type pointsResult struct {
	Data struct {
		UserID       int
		TotalScore   int
		TaskProgress []struct {
			Title        string
			CurrentScore int64
			DayMaxScore  int
		}
	}
}

func getPoints(ctx context.Context, print bool) (res *pointsResult, err error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, pointsLimit)
	defer cancel()

	done := chrome.ListenEvent(ctx, pointsAPI, "GET", true)
	if err = chromedp.Run(ctx, chromedp.Navigate(pointsURL)); err != nil {
		return
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case e := <-done:
		err = e.Unmarshal(&res)
	}
	if err != nil {
		log.Println("获取学习积分失败:", err)
	}
	if print {
		log.Print(res)
	}

	return
}

func (res *pointsResult) CreateTask() (task, *status) {
	t := task{true, true, articleNumber, videoNumber}
	for _, i := range res.Data.TaskProgress {
		switch i.Title {
		case "每日答题":
			if i.CurrentScore == 5 {
				t.practice = false
			}
		case "专项答题":
			if i.CurrentScore > 1 {
				t.paper = false
			}
		case "我要选读文章":
			if i.CurrentScore < 6 {
				t.article = articleNumber
			} else {
				t.article = articleNumber - (i.CurrentScore-6)*2
			}
		case "视听学习时长":
			t.video = videoNumber - i.CurrentScore*2
		}
	}
	return t, newStatus(t.article, t.video)
}

func (res pointsResult) String() string {
	m := make(map[string]string)
	for _, i := range res.Data.TaskProgress {
		m[i.Title] = fmt.Sprintf("%d分/%d分", i.CurrentScore, i.DayMaxScore)
	}

	output := fmt.Sprintf("用户ID: %d\n当前积分: %d\n", res.Data.UserID, res.Data.TotalScore)
	for _, i := range []string{"登录", "每日答题", "专项答题", "我要选读文章", "视听学习", "视听学习时长"} {
		output += fmt.Sprintf("%s: %s\n", i, m[i])
	}

	return output
}
