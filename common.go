package main

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func listenFetch(ctx context.Context, actions ...chromedp.Action) error {
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				c := chromedp.FromContext(ctx)
				ctx := cdp.WithExecutor(ctx, c.Target)

				if ev.ResourceType == network.ResourceTypeDocument ||
					ev.ResourceType == network.ResourceTypeScript ||
					ev.ResourceType == network.ResourceTypeStylesheet ||
					ev.ResourceType == network.ResourceTypeXHR {
					//log.Println("allow:", ev.Request.URL)
					fetch.ContinueRequest(ev.RequestID).Do(ctx)
				} else {
					//log.Println("block:", ev.Request.URL)
					fetch.FailRequest(ev.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
				}
			}()
		}
	})

	return chromedp.Run(ctx, append([]chromedp.Action{fetch.Enable()}, actions...)...)
}

type event struct {
	id    network.RequestID
	url   string
	bytes []byte
}

func listenURL(ctx context.Context, url string, method string, download bool) <-chan event {
	c, done := make(chan event, 1), make(chan event, 1)
	var m sync.Map
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if strings.HasPrefix(ev.Request.URL, url) && method == ev.Request.Method {
				m.Store(ev.RequestID, ev.Request.URL)
			}
		case *network.EventLoadingFinished:
			if v, ok := m.Load(ev.RequestID); ok {
				done <- event{ev.RequestID, v.(string), nil}
			}
		}
	})

	go func() {
		for {
			var event event
			select {
			case <-ctx.Done():
				return
			case event = <-done:
			}

			if download {
				if err := chromedp.Run(
					ctx,
					chromedp.ActionFunc(func(ctx context.Context) (err error) {
						event.bytes, err = network.GetResponseBody(event.id).Do(ctx)
						return
					}),
				); err != nil {
					log.Print(err)
				}
			}

			c <- event
		}
	}()

	return c
}

const pclogURL = "https://iflow-api.xuexi.cn/logflow/api/v1/pclog"

func listenPclog(ctx context.Context) <-chan struct{} {
	done, c := make(chan struct{}, 1), listenURL(ctx, pclogURL, "POST", false)
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
