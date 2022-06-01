package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slices"
)

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
	switch len(tips) {
	case 0:
		return nil, false
	case 1:
		log.Print("单选题")
		return []string{calcSingleChoice(choices, tips[0].NodeValue)}, false
	default:
		log.Print("多选题")
		return calcMultipleChoice(ctx, choices, tips)
	}
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
