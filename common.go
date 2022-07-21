package main

import (
	"context"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/sunshineplan/chrome"
)

func enableFetch(ctx context.Context) error {
	return chrome.EnableFetch(ctx, func(ev *fetch.EventRequestPaused) bool {
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
	})
}

const pclogURL = "https://iflow-api.xuexi.cn/logflow/api/v1/pclog"

func listenPclog(ctx context.Context) <-chan struct{} {
	done, c := make(chan struct{}, 1), chrome.ListenURL(ctx, pclogURL, "POST", false)
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
