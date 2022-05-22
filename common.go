package main

import (
	"context"
	"strings"

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

func listenURL(ctx context.Context, url string) <-chan []byte {
	c, done := make(chan []byte, 1), make(chan struct{}, 1)
	var id network.RequestID
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if strings.HasPrefix(ev.Request.URL, url) {
				id = ev.RequestID
			}
		case *network.EventLoadingFinished:
			if ev.RequestID == id {
				done <- struct{}{}
			}
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
			}

			var b []byte
			if err := chromedp.Run(
				ctx,
				chromedp.ActionFunc(func(ctx context.Context) (err error) {
					b, err = network.GetResponseBody(id).Do(ctx)
					return
				}),
			); err == nil {
				c <- b
			}
		}
	}()

	return c
}

func listenPclog(ctx context.Context) <-chan struct{} {
	done, c := make(chan struct{}, 1), listenURL(ctx, pclogURL)
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
