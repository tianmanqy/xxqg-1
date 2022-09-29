package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"
)

const (
	practiceURL = "https://pc.xuexi.cn/points/exam-practice.html"
	weeklyURL   = "https://pc.xuexi.cn/points/exam-weekly-list.html"
	paperURL    = "https://pc.xuexi.cn/points/exam-paper-list.html"

	weeklyClass = "week"
	paperClass  = "item"
)

var (
	token = flag.String("token", "", "token")
	force = flag.Bool("force", false, "force")
)

var tokenPath string

func init() {
	self, err := os.Executable()
	if err != nil {
		log.Println("Failed to get self path:", err)
	} else {
		tokenPath = filepath.Join(filepath.Dir(self), "xxqg.token")
	}
	rand.Seed(time.Now().UnixNano())
}

func main() {
	defer func() {
		fmt.Println("Press enter key to exit . . .")
		fmt.Scanln()
	}()

	iniflags.SetConfigFile(tokenPath)
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()

	c, err := login()
	if err != nil {
		log.Println("登录失败:", err)
		return
	}
	defer c.Close()

	res, err := getPoints(c)
	if err != nil {
		log.Println("获取学习积分失败:", err)
		if !*force {
			return
		}
	}
	log.Print(res)
	t := res.CreateTask()
	if t == emptyTask {
		log.Print("学习积分已达上限！")
		return
	}

	start := time.Now()

	dividingLine()
	for t.practice {
		checkError("每日答题", exam(c, practiceURL, ""))
		dividingLine()

		res, err = getPoints(c)
		if err != nil {
			log.Println("获取学习积分失败:", err)
			break
		}
		t = res.CreateTask()
	}
	if t.weekly {
		checkError("每周答题", exam(c, weeklyURL, weeklyClass))
		dividingLine()
	}
	if t.paper {
		checkError("专项答题", exam(c, paperURL, paperClass))
		dividingLine()
	}
	if t.article > 0 {
		checkError("选读文章", article(c, t.article))
		dividingLine()
	}
	if t.video > 0 {
		checkError("视听学习", video(c, t.video))
		dividingLine()
	}

	log.Printf("学习完成！总耗时：%s", time.Since(start))

	time.Sleep(time.Second)

	res, err = getPoints(c)
	if err != nil {
		log.Println("获取学习积分失败:", err)
	} else {
		log.Print(res)
	}
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
	io.WriteString(log.Default().Writer(), strings.Repeat("=", 100)+"\r\n")
}
