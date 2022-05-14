package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36"

var tokenPath string

func init() {
	self, err := os.Executable()
	if err != nil {
		log.Println("Failed to get self path:", err)
	} else {
		tokenPath = filepath.Join(filepath.Dir(self), "xxqg.token")
	}
}

func login() (context.Context, context.CancelFunc, error) {
	if *token != "" {
		ctx, cancel, err := loginWithToken()
		if err != nil {
			log.Printf("Token(%s)登录失败: %s", *token, err)
		} else {
			return ctx, cancel, nil
		}
	}

	return loginWithQRCode()
}

func loginWithQRCode() (context.Context, context.CancelFunc, error) {
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))...,
	)
	ctx, cancel = chromedp.NewContext(ctx)

	if err := listenFetch(ctx); err != nil {
		cancel()
		return nil, nil, err
	}

	loginCtx, loginCancel := context.WithTimeout(ctx, loginLimit)
	defer loginCancel()

	log.Print("请先扫码登录")
	if err := chromedp.Run(
		loginCtx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible("span.refresh"),
		chromedp.EvaluateAsDevTools(`$("span.refresh").scrollIntoViewIfNeeded()`, nil),
		chromedp.WaitVisible("span.logged-text"),
		getToken(),
	); err != nil {
		cancel()
		return nil, nil, err
	}
	log.Print("扫码登录成功")

	os.WriteFile(tokenPath, []byte("token="+*token), 0644)

	return ctx, cancel, nil
}

func loginWithToken() (context.Context, context.CancelFunc, error) {
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.UserAgent(ua))...,
	)
	ctx, cancel = chromedp.NewContext(ctx)

	if err := listenFetch(ctx); err != nil {
		cancel()
		return nil, nil, err
	}

	loginCtx, loginCancel := context.WithTimeout(ctx, tokenLimit)
	defer loginCancel()

	if err := chromedp.Run(
		loginCtx,
		network.SetCookie("token", *token).WithDomain(".xuexi.cn"),
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible("span.logged-text"),
	); err != nil {
		cancel()
		return nil, nil, err
	}
	log.Print("使用Token登录成功")

	return ctx, cancel, nil
}

func getToken() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		cookies, err := network.GetAllCookies().Do(ctx)
		if err != nil {
			return err
		}
		for _, cookie := range cookies {
			if cookie.Name == "token" {
				*token = cookie.Value
			}
		}
		return nil
	})
}
