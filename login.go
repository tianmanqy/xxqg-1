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
	tokenLimit = 2 * time.Second
)

func login() (chrome *chrome.Chrome, err error) {
	if *token != "" {
		if chrome, err = loginWithToken(); err != nil {
			log.Printf("Token(%s)登录失败: %s", *token, err)
		} else {
			log.Print("使用Token登录成功")
			return
		}
	}

	log.Print("请先扫码登录")
	if err = loginWithQRCode(); err != nil {
		return
	}
	log.Print("扫码登录成功")

	return loginWithToken()
}

func loginWithToken() (*chrome.Chrome, error) {
	chrome := newChrome(false, true)

	ctx, cancel := context.WithTimeout(chrome, loginLimit)
	defer cancel()

	if err := chromedp.Run(
		ctx,
		network.SetCookie("token", *token).WithDomain(".xuexi.cn"),
		chromedp.Navigate(loginURL),
		chromedp.WaitReady("div.login"),
	); err != nil {
		return nil, err
	}

	ctx, cancel = context.WithTimeout(ctx, tokenLimit)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.WaitVisible("span.logged-text")); err != nil {
		return nil, errors.New("无效Token")
	}

	return chrome, nil
}

func loginWithQRCode() error {
	chrome := newChrome(true, true)
	defer chrome.Close()

	ctx, cancel := context.WithTimeout(chrome, loginLimit)
	defer cancel()

	return chromedp.Run(
		ctx,
		chromedp.Navigate(loginURL),
		chromedp.ScrollIntoView("span.refresh", chromedp.NodeVisible),
		chromedp.WaitVisible("span.logged-text"),
		saveToken(),
	)
}

func saveToken() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		cookies, err := network.GetCookies().Do(ctx)
		if err != nil {
			return err
		}
		for _, cookie := range cookies {
			if cookie.Name == "token" {
				*token = cookie.Value
				return os.WriteFile(tokenPath, []byte("token="+*token), 0644)
			}
		}
		return errors.New("未找到Token")
	})
}
