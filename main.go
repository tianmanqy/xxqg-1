package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vharitonsky/iniflags"
)

var (
	token = flag.String("token", "", "token")
	force = flag.Bool("force", false, "force")
)

var tokenPath string

func main() {
	defer func() {
		//if err := recover(); err != nil {
		//	log.Print(err)
		//}

		fmt.Println("Press enter key to exit . . .")
		fmt.Scanln()
	}()

	self, err := os.Executable()
	if err != nil {
		log.Println("Failed to get self path:", err)
	} else {
		tokenPath = filepath.Join(filepath.Dir(self), "xxqg.token")
	}
	iniflags.SetConfigFile(tokenPath)
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()

	chrome, err := login()
	if err != nil {
		log.Println("登录失败:", err)
		return
	}
	defer chrome.Close()

	res, err := getPoints(chrome, true)
	if err != nil && !*force {
		return
	}

	task, status := res.CreateTask()
	if task == emptyTask {
		log.Print("学习积分已达上限！")
		return
	}
	defer status.Done()

	if task.article > 0 || task.video > 0 {
		chrome := newChrome(false, false)
		defer chrome.Close()
		go getItems(chrome, status)
	}

	start := time.Now()

	task.exam(chrome)
	res, err = task.study(chrome, status)
	dividingLine()
	log.Printf("学习完成！总耗时：%s", time.Since(start))
	if err == nil {
		if res == nil {
			if res, err = getPoints(chrome, false); err != nil {
				log.Print(err)
				return
			}
		}
		log.Print(res)
	}
}
