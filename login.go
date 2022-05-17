package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var tokenPath string

func init() {
	self, err := os.Executable()
	if err != nil {
		log.Println("Failed to get self path:", err)
	} else {
		tokenPath = filepath.Join(filepath.Dir(self), "xxqg.token")
	}
}

func login(ctx context.Context) error {
	if *token != "" {
		if err := loginWithToken(ctx); err != nil {
			log.Printf("Token(%s)登录失败: %s", *token, err)
		} else {
			return nil
		}
	}

	return loginWithQRCode(ctx)
}

func loginWithQRCode(ctx context.Context) error {
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
		return err
	}
	log.Print("扫码登录成功")

	os.WriteFile(tokenPath, []byte("token="+*token), 0644)

	return nil
}

func loginWithToken(ctx context.Context) error {
	loginCtx, loginCancel := context.WithTimeout(ctx, tokenLimit)
	defer loginCancel()

	if err := chromedp.Run(
		loginCtx,
		network.SetCookie("token", *token).WithDomain(".xuexi.cn"),
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible("span.logged-text"),
	); err != nil {
		return err
	}
	log.Print("使用Token登录成功")

	return nil
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
