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

func login() (*chrome.Chrome, error) {
	c := chrome.Headful(false)
	if *token != "" {
		if err := loginWithToken(c); err != nil {
			log.Printf("Token(%s)登录失败: %s", *token, err)
		} else {
			return c, nil
		}
	}
	if err := loginWithQRCode(c); err != nil {
		return nil, err
	}

	return c, nil
}

func loginWithQRCode(c *chrome.Chrome) error {
	if err := c.EnableFetch(filter); err != nil {
		return err
	}

	loginCtx, loginCancel := context.WithTimeout(c, loginLimit)
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
		c.Close()
		return err
	}
	log.Print("扫码登录成功")

	os.WriteFile(tokenPath, []byte("token="+*token), 0644)

	return nil
}

func loginWithToken(c *chrome.Chrome) error {
	if err := c.EnableFetch(filter); err != nil {
		return err
	}

	loginCtx, loginCancel := context.WithTimeout(c, loginLimit)
	defer loginCancel()

	if err := chromedp.Run(
		loginCtx,
		network.SetCookie("token", *token).WithDomain(".xuexi.cn"),
		chromedp.Navigate(loginURL),
		chromedp.WaitReady("div.login"),
	); err != nil {
		c.Close()
		return err
	}

	tokenCtx, tokenCancel := context.WithTimeout(loginCtx, tokenLimit)
	defer tokenCancel()

	if err := chromedp.Run(tokenCtx, chromedp.WaitVisible("span.logged-text")); err != nil {
		c.Close()
		return errors.New("无效Token")
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
