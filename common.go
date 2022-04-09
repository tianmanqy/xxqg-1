package main

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func listenFetch(ctx context.Context) error {
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

	return chromedp.Run(ctx, fetch.Enable())
}

func listenPclog(ctx context.Context) <-chan struct{} {
	done, c := make(chan struct{}, 1), make(chan struct{}, 1)
	var n int
	go func() {
		for {
			select {
			case <-c:
				if n == 1 {
					done <- struct{}{}
					n = 0
				} else {
					n++
				}
			case <-ctx.Done():
				close(done)
				return
			}
		}
	}()

	var id network.RequestID
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if ev.Request.URL == pclogURL {
				id = ev.RequestID
			}
		case *network.EventLoadingFinished:
			if ev.RequestID == id {
				c <- struct{}{}
			}
		}
	})

	return done
}
