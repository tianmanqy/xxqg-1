package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
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
		var choices, tips []*cdp.Node
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

		var incalculable bool
		if len(inputs) == 0 {
			var answers []string
			choices, answers, incalculable, err = getChoiceQuestionAnswers(ctx, tips)
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
						var str string
						if err = chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`$("div.q-body").innerText`, &str)); err != nil {
							return
						}
						str = randomString(str, rand.Intn(3)+2)
						log.Println("随机输入", str)
						if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, str)); err != nil {
							return
						}
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
				if len(inputs) == 0 && !incalculable {
					printTips(tips)
					printChoices(choices)
				}
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

func getChoiceQuestionAnswers(ctx context.Context, tips []*cdp.Node) (
	choices []*cdp.Node, answers []string, incalculable bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, choiceLimit)
	defer cancel()

	if err = chromedp.Run(
		ctx,
		chromedp.Nodes(fmt.Sprintf("//div[%s]/text()", classSelector("q-answer")), &choices, chromedp.AtLeast(0)),
	); err != nil {
		return
	}

	answers, incalculable = calcChoiceQuestion(ctx, choices, tips)

	return
}

func calcChoiceQuestion(ctx context.Context, choices []*cdp.Node, tips []*cdp.Node) ([]string, bool) {
	if len(tips) == 1 {
		log.Print("单选题")
		return []string{calcSingleChoice(choices, tips[0].NodeValue)}, false
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
		if !regexp.MustCompile(`^\s*[A-Z\.]\s*$`).MatchString(i.NodeValue) {
			diffs := diff.DiffMain(tip, i.NodeValue, false)
			res = append(res, result{i.NodeValue, diff.DiffLevenshtein(diffs)})
		} else {
			res = append(res, result{i.NodeValue, 100})
		}
	}
	slices.SortStableFunc(res, func(a, b result) bool { return a.distance < b.distance })
	return res[0].text
}

func calcMultipleChoice(ctx context.Context, choices []*cdp.Node, tips []*cdp.Node) (res []string, incalculable bool) {
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
		incalculable = true
		log.Print("无法计算答案")
		printTips(tips)
		printChoices(choices)
		if len(res) == 0 {
			res = []string{calcSingleChoice(choices, str)}
			return
		}
	case <-done:
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

func printChoices(nodes []*cdp.Node) {
	var str string
	for _, node := range nodes {
		str += node.NodeValue
	}
	log.Println("选项:", str)
}

func printTips(nodes []*cdp.Node) {
	var value []string
	for i, node := range nodes {
		value = append(value, fmt.Sprintf("%d. %s", i+1, node.NodeValue))
	}
	log.Println("提示:", strings.Join(value, " "))
}
