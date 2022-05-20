package main

import (
	"context"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var (
	headful = newChrome().
		addFlags(chromedp.Flag("headless", false)).
		addActions(addScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>false})"))
)

type chrome struct {
	flags   []chromedp.ExecAllocatorOption
	actions []chromedp.Action
}

func newChrome() *chrome {
	return &chrome{}
}

func (c *chrome) addFlags(flags ...chromedp.ExecAllocatorOption) *chrome {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *chrome) addActions(actions ...chromedp.Action) *chrome {
	c.actions = append(c.actions, actions...)
	return c
}

func (c *chrome) context() (context.Context, context.CancelFunc, error) {
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...,
	)
	ctx, cancel = chromedp.NewContext(ctx)

	if err := listenFetch(ctx, c.actions...); err != nil {
		cancel()
		return nil, nil, err
	}

	return ctx, cancel, nil
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
