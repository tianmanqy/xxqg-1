package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slices"
)

func getChoiceQuestionAnswers(ctx context.Context, body string, tips []*cdp.Node) (
	choices []*cdp.Node, answers []string, incalculable bool, err error,
) {
	ctx, cancel := context.WithTimeout(ctx, choiceLimit)
	defer cancel()

	if err = chromedp.Run(
		ctx,
		chromedp.Nodes(fmt.Sprintf("//div[%s]/text()", classSelector("q-answer")), &choices, chromedp.AtLeast(0)),
	); err != nil {
		return
	}

	n := strings.Count(body, "（）")
	switch n {
	case 0:
		log.Print("未知选题")
	case 1:
		log.Print("单选题")
		answers = []string{calcSingleChoice(choices, tips)}
		return
	default:
		log.Printf("多选题(%d)", n)
	}

	answers, incalculable = calcMultipleChoice(ctx, n, choices, tips)
	if incalculable {
		log.Print("无法计算答案")
		log.Println("题目:", body)
		printTips(tips)
		printChoices(choices)
	}

	return
}

var diff = diffmatchpatch.New()

func calcSingleChoice(choices, tips []*cdp.Node) string {
	tip := fullTips(tips)

	type result struct {
		text     string
		distance int
	}
	var res []result
	for i, choice := range choices {
		if i%3 == 2 {
			if choice.NodeValue == tip {
				return tip
			}
			res = append(res, result{choice.NodeValue, diff.DiffLevenshtein(diff.DiffMain(tip, choice.NodeValue, false))})
		}
	}
	slices.SortStableFunc(res, func(a, b result) bool { return a.distance < b.distance })

	return res[0].text
}

func calcMultipleChoice(ctx context.Context, n int, choices, tips []*cdp.Node) (res []string, incalculable bool) {
	if n == len(choices)/3 {
		for i, choice := range choices {
			if i%3 == 2 {
				res = append(res, choice.NodeValue)
			}
		}
		return
	}

	tip := fullTips(tips)
	str := tip
	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			if str == "" {
				return
			}
			for i, choice := range choices {
				if i%3 == 2 {
					if strings.HasPrefix(str, choice.NodeValue) {
						res = append(res, choice.NodeValue)
						str = strings.Replace(str, choice.NodeValue, "", 1)
						continue
					}
					if strings.ReplaceAll(choice.NodeValue, " ", "") == tip {
						res = []string{choice.NodeValue}
						return
					}
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		incalculable = true
		if len(res) == 0 {
			res = []string{calcSingleChoice(choices, tips)}
		}
	case <-done:
	}

	return
}

func fullTips(tips []*cdp.Node) (str string) {
	for _, i := range tips {
		str += i.NodeValue
	}
	return
}
