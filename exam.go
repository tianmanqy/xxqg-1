package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
)

const (
	weeklyAPI = "https://pc-proxy-api.xuexi.cn/api/exam/service/practice/pc/weekly/more"
	paperAPI  = "https://pc-proxy-api.xuexi.cn/api/exam/service/paper/pc/list"
	scoreAPI  = "https://pc-proxy-api.xuexi.cn/api/exam/service/detail/score"

	examLimit = 15 * time.Second
)

func exam(ctx context.Context, url, class string) (err error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	if err = chromedp.Run(ctx); err != nil {
		return
	}

	navCtx, cancel := context.WithTimeout(ctx, examLimit)
	defer cancel()

	if err = chromedp.Run(navCtx, chromedp.Navigate(url)); err != nil {
		return
	}

	var title string
	if class == "" {
		title = "每日答题"
	} else {
		var page int
		var buttons []*cdp.Node
		page, err = getPageNumber(navCtx)
		if err != nil {
			return err
		}

		pageCtx, cancel := context.WithTimeout(ctx, time.Duration(page)*time.Second)
		defer cancel()

		var api string
		switch class {
		case weeklyClass:
			api = weeklyAPI
		case paperClass:
			api = paperAPI
		}
		more := chrome.ListenEvent(pageCtx, api, "GET", false)
		for i := 0; i < page; i++ {
			if err = chromedp.Run(
				pageCtx,
				chromedp.WaitVisible("div.ant-spin-container"),
				chromedp.Nodes(fmt.Sprintf("div.%s button:not(.ant-btn-background-ghost)", class), &buttons, chromedp.AtLeast(0)),
			); err != nil {
				return
			}

			if len(buttons) != 0 || i == page-1 {
				break
			} else {
				time.Sleep(200 * time.Millisecond)
				if err = chromedp.Run(pageCtx, chromedp.Click(`li[title="Next Page"][aria-disabled=false]`)); err != nil {
					return
				}

				select {
				case <-pageCtx.Done():
					return pageCtx.Err()
				case <-more:
				}
			}
		}

		if len(buttons) == 0 {
			return fmt.Errorf("没有可用试题")
		}

		if err = chromedp.Run(
			pageCtx,
			chromedp.MouseClickNode(buttons[0]),
			chromedp.WaitVisible("div.question"),
			chromedp.Text("div.title", &title, chromedp.AtLeast(0)),
		); err != nil {
			return
		}
	}

	countCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	n, err := getExamNumber(countCtx)
	if err != nil {
		return
	}
	log.Printf("[答题] %s(共%d题)", title, n)

	ctx, cancel = context.WithTimeout(ctx, time.Duration(n)*examLimit)
	defer cancel()

	start := time.Now()
	done := chrome.ListenEvent(ctx, scoreAPI, "GET", false)
	go func() {
		for i := 1; i <= n; i++ {
			log.Printf("#题目%d", i)
			var tips, inputs, choices []*cdp.Node
			var body, tip string
			if err = chromedp.Run(
				ctx,
				chromedp.Click("span.tips", chromedp.NodeVisible),
				chromedp.EvaluateAsDevTools(`$("div.line-feed").innerText`, &tip),
				chromedp.Nodes(`//div[@class="line-feed"]//font[@color="red"]/text()`, &tips, chromedp.AtLeast(0)),
				chromedp.Click("div.q-header>svg"),
				chromedp.WaitNotVisible("div.line-feed"),
				chromedp.Nodes("input.blank", &inputs, chromedp.AtLeast(0)),
				chromedp.EvaluateAsDevTools(`$("div.q-body").innerText`, &body),
			); err != nil {
				return
			}
			if len(tips) == 0 {
				log.Print("没有提示答案")
			}

			var answers []string
			var incalculable bool
			if len(inputs) == 0 {
				choices, answers, incalculable, err = getChoiceQuestionAnswers(ctx, body, tip, tips)
				if err != nil {
					return
				}

				if len(answers) != 0 {
					for _, i := range answers {
						log.Println("选择", i)
						if err = chromedp.Run(
							ctx,
							chromedp.Sleep(time.Second),
							chromedp.Click(fmt.Sprintf("//div[%s][text()=%q]", classSelector("q-answer"), i)),
						); err != nil {
							return
						}
					}
				} else {
					log.Print("未找到选择题答案")
					if err = chromedp.Run(ctx, chromedp.Click(fmt.Sprintf("//div[%s]", classSelector("q-answer")))); err != nil {
						return
					}
				}
			} else {
				log.Print("填空题")
				if len(inputs) == 1 && len(tips) > 1 {
					var str string
					for _, i := range tips {
						str += i.NodeValue
					}
					log.Println("合并输入", str)
					if err = chromedp.Run(ctx, chromedp.KeyEventNode(inputs[0], str)); err != nil {
						return
					}
				} else {
					for i, input := range inputs {
						if i < len(tips) {
							log.Println("输入", tips[i].NodeValue)
							if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, tips[i].NodeValue)); err != nil {
								return
							}
						} else {
							str := randomString(body, rand.Intn(3)+2)
							log.Println("随机输入", str)
							if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, str)); err != nil {
								return
							}
						}
					}
				}
			}

			if class == paperClass && i == n {
				if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.submit-btn", chromedp.NodeEnabled)); err != nil {
					return
				}
			} else {
				if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.next-btn", chromedp.NodeEnabled)); err != nil {
					return
				}

				var nodes []*cdp.Node
				if err = chromedp.Run(
					ctx,
					chromedp.Sleep(time.Second),
					chromedp.Nodes("div.action-row>button.next-btn:enabled", &nodes, chromedp.AtLeast(0)),
				); err != nil {
					return
				}

				if len(nodes) != 0 {
					log.Print("答错 ×")
					if len(inputs) == 0 && !incalculable {
						log.Println("题目:", body)
						log.Println("提示:", tip)
						printTips(tips)
						printChoices(choices)
					}
					var answer string
					if err := chromedp.Run(
						ctx,
						chromedp.EvaluateAsDevTools(`$("div.answer").innerText`, &answer),
						chromedp.Click("div.action-row>button.next-btn"),
					); err != nil {
						log.Println("无法获取答案:", err)
					} else {
						log.Print(answer)
					}
				} else {
					if class != paperClass {
						log.Print("答对 √")
					}
				}
			}

			time.Sleep(2 * time.Second)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	log.Println("答题完毕！耗时:", time.Since(start))
	return
}

func getPageNumber(ctx context.Context) (n int, err error) {
	var buttons []*cdp.Node
	if err = chromedp.Run(ctx, chromedp.Nodes("li.ant-pagination-item", &buttons)); err != nil {
		return
	}

	if length := len(buttons); length == 0 {
		err = fmt.Errorf("no pagination item found")
	} else {
		n, err = strconv.Atoi(buttons[length-1].AttributeValue("title"))
	}

	return
}

func getExamNumber(ctx context.Context) (n int, err error) {
	var pager string
	if err = chromedp.Run(
		ctx,
		chromedp.WaitVisible("div.question"),
		chromedp.EvaluateAsDevTools(`$("div.pager").innerText`, &pager),
	); err != nil {
		return
	}

	if res := regexp.MustCompile(`\d+/(\d+)`).FindStringSubmatch(pager); len(res) == 2 {
		n, err = strconv.Atoi(res[1])
	} else {
		err = fmt.Errorf("获取题目数量失败: %s", pager)
	}

	return
}

func classSelector(class string) string {
	return fmt.Sprintf(`contains(concat(" ", normalize-space(@class), " "), " %s ")`, class)
}

func randomString(str string, size int) string {
	rs := []rune(str)
	if length := len(rs); length > size {
		n := rand.Intn(length - size)
		str = string(rs[n : n+size])
	}
	str = regexp.MustCompile("[。？！，、；：“”‘’'（）《》〈〉【】『』「」﹃﹄〔〕…—～﹏￥]").ReplaceAllString(str, "")
	if str == "" {
		return "不知道"
	}
	return str
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
