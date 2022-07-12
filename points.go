package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	pointsURL = "https://pc.xuexi.cn/points/my-points.html"
	pointsAPI = "https://pc-proxy-api.xuexi.cn/api/score/days/listScoreProgress"

	pointsLimit = 15 * time.Second

	articleNumber = 12
	videoNumber   = 12
)

type task struct {
	practice, weekly, paper bool
	article, video          int
}

type pointsResult struct {
	Data struct {
		UserID       int
		TotalScore   int
		TaskProgress []struct {
			Title        string
			CurrentScore int
			DayMaxScore  int
		}
	}
}

func getPoints(ctx context.Context) (res pointsResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, pointsLimit)
	defer cancel()

	done := listenURL(ctx, pointsAPI, "GET", true)
	if err = chromedp.Run(ctx, chromedp.Navigate(pointsURL)); err != nil {
		return
	}

	var event event
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return
	case event = <-done:
	}

	err = json.Unmarshal(event.bytes, &res)

	return
}

func (res pointsResult) CreateTask() task {
	t := task{true, true, true, articleNumber, videoNumber}
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
				t.article = articleNumber
			} else {
				t.article = articleNumber - (i.CurrentScore-6)*2
			}
		case "视听学习时长":
			t.video = videoNumber - i.CurrentScore*2
		}
	}
	return t
}

func (res pointsResult) String() string {
	m := make(map[string]string)
	for _, i := range res.Data.TaskProgress {
		m[i.Title] = fmt.Sprintf("%d分/%d分", i.CurrentScore, i.DayMaxScore)
	}

	output := fmt.Sprintf("用户ID: %d\n当前积分: %d\n", res.Data.UserID, res.Data.TotalScore)
	for _, i := range []string{"登录", "每日答题", "每周答题", "专项答题", "我要选读文章", "视听学习", "视听学习时长"} {
		output += fmt.Sprintf("%s: %s\n", i, m[i])
	}

	return output
}
