package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/sunshineplan/chrome"
)

const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36"

func newChrome(headful, disableAutomationControlled bool) (c *chrome.Chrome) {
	if headful {
		c = chrome.Headful()
	} else {
		c = chrome.Headless().UserAgent(ua)
	}
	if disableAutomationControlled {
		c.DisableAutomationControlled()
	}
	if err := c.EnableFetch(func(ev *fetch.EventRequestPaused) bool {
		return ev.ResourceType == network.ResourceTypeDocument ||
			ev.ResourceType == network.ResourceTypeScript ||
			ev.ResourceType == network.ResourceTypeStylesheet ||
			ev.ResourceType == network.ResourceTypeXHR
	}); err != nil {
		c.Close()
		panic(err)
	}
	return
}

func classSelector(class string) string {
	return fmt.Sprintf(`contains(concat(" ", normalize-space(@class), " "), " %s ")`, class)
}

func printChoices(choices []*cdp.Node) (output string) {
	for i, choice := range choices {
		switch i % 3 {
		case 0:
			if i != 0 {
				output += " "
			}
			output += choice.NodeValue + "."
		case 2:
			output += choice.NodeValue
		}
	}
	output = "选项: " + output

	log.Print(output)
	return
}

func printTips(tips []*cdp.Node) (output string) {
	var value []string
	for i, tip := range tips {
		value = append(value, fmt.Sprintf("%d.%s", i+1, tip.NodeValue))
	}
	output = "提示项: " + strings.Join(value, " ")

	log.Print(output)
	return
}
