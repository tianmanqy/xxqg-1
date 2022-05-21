package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slices"
)

func exam(ctx context.Context, url, class string, n int, d time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	if err = chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		return
	}

	if class != "" {
		var buttons []*cdp.Node
		if err = chromedp.Run(
			ctx,
			chromedp.WaitVisible("div.month"),
			chromedp.Sleep(time.Second),
			chromedp.Nodes(fmt.Sprintf("div.%s button:not(.ant-btn-background-ghost)", class), &buttons, chromedp.AtLeast(0)),
		); err != nil {
			return
		}
		for {
			if len(buttons) != 0 {
				break
			}
			if err = chromedp.Run(
				ctx,
				chromedp.Click(`li[title="Next Page"][aria-disabled=false]`),
				chromedp.WaitVisible("div.month"),
				chromedp.Sleep(time.Second),
				chromedp.Nodes(fmt.Sprintf("div.%s button:not(.ant-btn-background-ghost)", class), &buttons, chromedp.AtLeast(0)),
			); err != nil {
				return
			}
		}

		var title string
		if err = chromedp.Run(
			ctx,
			chromedp.MouseClickNode(buttons[0]),
			chromedp.WaitVisible("div.question"),
			chromedp.Text("div.title", &title, chromedp.AtLeast(0)),
		); err != nil {
			return
		}

		log.Println("[答题]", title)
	} else {
		log.Println("[答题] 每日答题")
	}

	start := time.Now()
	for i := 1; i <= n; i++ {
		log.Printf("#题目%d", i)
		var tips []*cdp.Node
		if err = chromedp.Run(
			ctx,
			chromedp.Click("span.tips", chromedp.NodeVisible),
			chromedp.Nodes(`//div[@class="line-feed"]//font[@color="red"]/text()`, &tips, chromedp.AtLeast(0)),
			chromedp.Click("div.q-header>svg"),
			chromedp.WaitNotVisible("div.line-feed"),
		); err != nil {
			return
		}
		if len(tips) == 0 {
			log.Print("没有提示答案")
		}

		var inputs []*cdp.Node
		if err = chromedp.Run(ctx, chromedp.Nodes("input.blank", &inputs, chromedp.AtLeast(0))); err != nil {
			return
		}

		if len(inputs) == 0 {
			var answers []string
			answers, err = getChoiceQuestionAnswers(ctx, tips)
			if err != nil {
				return
			}

			if len(answers) != 0 {
				for _, i := range answers {
					time.Sleep(time.Second)

					log.Println("选择", i)

					if err = chromedp.Run(
						ctx,
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
			for i, input := range inputs {
				if len(inputs) == len(tips) {
					log.Println("输入", tips[i].NodeValue)
					if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, tips[i].NodeValue)); err != nil {
						return
					}
				} else {
					log.Println("输入", "不知道")
					if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, "不知道")); err != nil {
						return
					}
				}
			}
		}

		time.Sleep(time.Second)

		if i == 10 {
			if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.submit-btn", chromedp.NodeEnabled)); err != nil {
				return
			}
		} else {
			if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.next-btn", chromedp.NodeEnabled)); err != nil {
				return
			}

			time.Sleep(2 * time.Second)

			var nodes []*cdp.Node
			if err = chromedp.Run(
				ctx,
				chromedp.Nodes("div.action-row>button.next-btn:enabled", &nodes, chromedp.AtLeast(0)),
			); err != nil {
				return
			}

			if len(nodes) != 0 {
				log.Print("答错 ×")
				if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.next-btn")); err != nil {
					return
				}
			} else {
				if class != paperClass {
					log.Print("答对 √")
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	log.Printf("答题完毕！耗时：%s", time.Since(start))
	return
}

func getChoiceQuestionAnswers(ctx context.Context, tips []*cdp.Node) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var choices []*cdp.Node
	if err := chromedp.Run(
		ctx,
		chromedp.Nodes(fmt.Sprintf("//div[%s]/text()", classSelector("q-answer")), &choices, chromedp.AtLeast(0)),
	); err != nil {
		return nil, err
	}

	return calcChoiceQuestion(ctx, choices, tips), nil
}

func calcChoiceQuestion(ctx context.Context, choices []*cdp.Node, tips []*cdp.Node) []string {
	if len(tips) == 1 {
		log.Print("单选题")
		return []string{calcSingleChoice(choices, tips[0].NodeValue)}
	}

	log.Print("多选题")
	return calcMultipleChoice(ctx, choices, tips)
}

var diff = diffmatchpatch.New()

func calcSingleChoice(choices []*cdp.Node, tip string) string {
	type result struct {
		text     string
		distance int
	}
	var res []result
	for _, i := range choices {
		if i.NodeValue == tip {
			return tip
		}
		diffs := diff.DiffMain(tip, i.NodeValue, false)
		res = append(res, result{i.NodeValue, diff.DiffLevenshtein(diffs)})
	}
	slices.SortStableFunc(res, func(a, b result) bool { return a.distance < b.distance })
	return res[0].text
}

func calcMultipleChoice(ctx context.Context, choices []*cdp.Node, tips []*cdp.Node) (res []string) {
	var str string
	for _, i := range tips {
		str += i.NodeValue
	}
	fullstr := str

	done := make(chan struct{})
	go func() {
		for {
			if str == "" {
				close(done)
				return
			}
			for _, i := range choices {
				if strings.HasPrefix(str, i.NodeValue) {
					res = append(res, i.NodeValue)
					str = strings.Replace(str, i.NodeValue, "", 1)
					continue
				}
				if fullstr == strings.ReplaceAll(i.NodeValue, " ", "") {
					res = []string{i.NodeValue}
					str = ""
					break
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Print("无法计算答案")
		for _, i := range tips {
			log.Println("tips:", i.NodeValue)
		}
		for _, i := range choices {
			log.Println("answers:", i.NodeValue)
		}
		if len(res) == 0 {
			for _, i := range choices {
				if strings.Contains(i.NodeValue, str) {
					res = append(res, i.NodeValue)
					return
				}
			}
		}
	case <-done:
	}

	return
}

func classSelector(class string) string {
	return fmt.Sprintf(`contains(concat(" ", normalize-space(@class), " "), " %s ")`, class)
}
