package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/sunshineplan/chrome"
)

func filter(ev *fetch.EventRequestPaused) bool {
	res := ev.ResourceType == network.ResourceTypeDocument ||
		ev.ResourceType == network.ResourceTypeScript ||
		ev.ResourceType == network.ResourceTypeStylesheet ||
		ev.ResourceType == network.ResourceTypeXHR
	if res {
		//log.Println("allow:", ev.Request.URL)
	} else {
		//log.Println("block:", ev.Request.URL)
	}
	return res
}

const pclogURL = "https://iflow-api.xuexi.cn/logflow/api/v1/pclog"

func listenPclog(ctx context.Context) <-chan struct{} {
	done, c := make(chan struct{}, 1), chrome.ListenEvent(ctx, pclogURL, "POST", false)
	var n int
	go func() {
		for {
			select {
			case <-c:
				if n%2 == 1 {
					done <- struct{}{}
				}
				n++
			case <-ctx.Done():
				return
			}
		}
	}()

	return done
}

func classSelector(class string) string {
	return fmt.Sprintf(`contains(concat(" ", normalize-space(@class), " "), " %s ")`, class)
}

func printChoices(choices []*cdp.Node) (output string) {
	for i, choice := range choices {
		switch i % 3 {
		case 0:
			if i != 0 {
				output += " "
			}
			output += choice.NodeValue + "."
		case 2:
			output += choice.NodeValue
		}
	}
	output = "选项: " + output

	log.Print(output)
	return
}

func printTips(tips []*cdp.Node) (output string) {
	var value []string
	for i, tip := range tips {
		value = append(value, fmt.Sprintf("%d.%s", i+1, tip.NodeValue))
	}
	output = "提示项: " + strings.Join(value, " ")

	log.Print(output)
	return
}
