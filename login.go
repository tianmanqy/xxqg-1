package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
)

const (
	loginURL = "https://pc.xuexi.cn/points/login.html"

	loginLimit = 2 * time.Minute
	tokenLimit = time.Second
)

func login() (context.Context, context.CancelFunc, error) {
	if *token != "" {
		ctx, cancel, err := loginWithToken(chrome.Headful(false))
		if err != nil {
			log.Printf("Token(%s)登录失败: %s", *token, err)
		} else {
			return ctx, cancel, nil
		}
	}

	return loginWithQRCode(chrome.Headful(false))
}

func loginWithQRCode(c *chrome.Chrome) (context.Context, context.CancelFunc, error) {
	ctx, cancel, err := c.Context()
	if err != nil {
		return nil, nil, err
	}

	if err := enableFetch(ctx); err != nil {
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

func loginWithToken(c *chrome.Chrome) (context.Context, context.CancelFunc, error) {
	ctx, cancel, err := c.Context()
	if err != nil {
		return nil, nil, err
	}

	if err := enableFetch(ctx); err != nil {
		return nil, nil, err
	}

	loginCtx, loginCancel := context.WithTimeout(ctx, loginLimit)
	defer loginCancel()

	if err := chromedp.Run(
		loginCtx,
		network.SetCookie("token", *token).WithDomain(".xuexi.cn"),
		chromedp.Navigate(loginURL),
		chromedp.WaitReady("div.login"),
	); err != nil {
		cancel()
		return nil, nil, err
	}

	tokenCtx, tokenCancel := context.WithTimeout(loginCtx, tokenLimit)
	defer tokenCancel()

	if err := chromedp.Run(tokenCtx, chromedp.WaitVisible("span.logged-text")); err != nil {
		cancel()
		return nil, nil, errors.New("无效Token")
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
