package main

import (
	"context"
	"io"
	"log"
	"strings"
)

const (
	practiceURL = "https://pc.xuexi.cn/points/exam-practice.html"
	paperURL    = "https://pc.xuexi.cn/points/exam-paper-list.html"

	paperClass = "item"
)

var emptyTask task

type task struct {
	practice, paper bool
	article, video  int64
}

func (task task) exam(ctx context.Context) {
	for task.practice {
		dividingLine()
		checkError("每日答题", exam(ctx, practiceURL, ""))

		if res, err := getPoints(ctx, false); err != nil {
			break
		} else {
			task, _ = res.CreateTask()
		}
	}
	if task.paper {
		dividingLine()
		checkError("专项答题", exam(ctx, paperURL, paperClass))
	}
}

func (task task) study(ctx context.Context, status *status) (res *pointsResult, err error) {
	var retry bool
	for task.article > 0 || task.video > 0 {
		dividingLine()
		if retry {
			log.Print("尚未达到目标积分，重新制定计划")
			retry = false
		}
		study(ctx, task.article, task.video, status)
		if res, err = getPoints(ctx, false); err != nil {
			return
		}
		task, _ = res.CreateTask()
		if task.article > 0 || task.video > 0 {
			status.addArticle(task.article)
			status.addVideo(task.video)
			retry = true
		}
	}
	return
}

func checkError(task string, err error) {
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Printf("%s: 任务超时", task)
		} else {
			log.Printf("%s: %s", task, err)
		}
	}
}

func dividingLine() {
	io.WriteString(log.Default().Writer(), strings.Repeat("=", 64)+"\r\n")
}
