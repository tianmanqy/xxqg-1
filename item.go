package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/sunshineplan/chrome"
	"github.com/sunshineplan/gohttp"
)

const utf8Space = " " // U+00a0

type itemType int

const (
	articleItem itemType = 1
	pictureItem itemType = 2
	videoItem   itemType = 30
	audioItem   itemType = 52
	specialItem itemType = 100
)

type item struct {
	local local
	index int
	Type  itemType
	date  string
	url   string
	title string
}

func parseItems(local local, url string, event *chrome.Event) (items []item, err error) {
	var channel struct {
		Items [30]struct {
			InsertTime string
			ItemTitle  string
			ItemType   itemType
		}
	}
	if err = event.Unmarshal(&channel); err != nil {
		if err = gohttp.Get(event.Response.Response.URL, nil).JSON(&channel); err != nil {
			return
		}
	}

	index := 1
	for _, i := range channel.Items {
		switch i.ItemType {
		case articleItem, pictureItem, videoItem, audioItem, specialItem:
			var date string
			if insertTime := strings.Fields(i.InsertTime); len(insertTime) > 0 {
				date = insertTime[0]
			}
			items = append(items, item{local, index, i.ItemType, date, url, strings.TrimSpace(i.ItemTitle)})
			index++
		}
	}
	return
}

func (i item) byIndex() any {
	return fmt.Sprintf("(//div[contains(@class,'list_text')]//a)[%d]", i.index)
}

func (i item) byTitle() any {
	var runes []rune
	for i, r := range i.title {
		if i < 60 {
			runes = append(runes, r)
		}
	}
	return fmt.Sprintf(`//a[contains(.,"%s")]`, strings.ReplaceAll(string(runes), " ", utf8Space))
}

func (i item) compare(title string) bool {
	return strings.HasPrefix(i.title, strings.ReplaceAll(strings.TrimSuffix(title, "…"), utf8Space, " "))
}

func (i item) String() string {
	return fmt.Sprintln(i.local, i.index, i.Type, i.title)
}

type items struct {
	articles chan item
	videos   chan item
}

var (
	locals = make(chan items, len(localList))
	allGet bool
)

func getItems(ctx context.Context, status *status) {
	defer func() {
		if err := recover(); err != nil {
			if msg, ok := err.(string); ok {
				log.Print(msg)
			}
			close(locals)
			close(articles)
			close(videos)
		}
	}()
	rand.Shuffle(len(localList), func(i, j int) { localList[i], localList[j] = localList[j], localList[i] })
	maxRetry := 5
	for _, local := range localList {
		if err := ctx.Err(); err != nil {
			panic(err)
		}

		select {
		case <-status.pause:
			select {
			case <-ctx.Done():
				panic(ctx.Err())
			case <-status.done:
				return
			case <-status.run:
			}
		case <-status.run:
		}
		if items, err := local.items(ctx, status, time.Now()); err != nil {
			log.Printf("无法获取%s的内容: %s", local, err)
			if maxRetry > 0 {
				maxRetry--
				continue
			}
			panic("获取内容重试次数已达上限")
		} else if len(items.articles) > 0 || len(items.videos) > 0 {
			locals <- items
		}
	}
	allGet = true
	panic("已获取所有内容")
}
