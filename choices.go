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

	var header string
	if err = chromedp.Run(
		ctx,
		chromedp.EvaluateAsDevTools(`$("div.q-header").innerText`, &header),
		chromedp.Nodes(fmt.Sprintf("//div[%s]/text()", classSelector("q-answer")), &choices, chromedp.AtLeast(0)),
	); err != nil {
		return
	}

	slice := convertNodes(choices, 3)
	n := strings.Count(body, "（）")
	switch header {
	case "单选题":
		log.Print(header)
		answers = []string{calcSingleChoice(ctx, slice, convertNodes(tips, 1))}
		return
	case "多选题":
		log.Printf("多选题(%d)", n)
	default:
		log.Printf("未知题型: %s(%d)", header, n)
	}

	if n == len(slice) {
		for _, choice := range slice {
			answers = append(answers, choice)
		}
		return
	}

	answers, _, incalculable = calcMultipleChoice(ctx, slice, convertNodes(tips, 1))
	if incalculable {
		log.Print("无法计算答案")
		log.Println("题目:", body)
		printTips(tips)
		printChoices(choices)
	}

	return
}

var diff = diffmatchpatch.New()

func calcSingleChoice(ctx context.Context, choices, tips []string) string {
	type result struct {
		text     string
		distance int
	}
	var res []result
	str := strings.Join(tips, "")
	for _, choice := range choices {
		if choice == str {
			return str
		}
		res = append(res, result{choice, diff.DiffLevenshtein(diff.DiffMain(str, choice, false))})
	}
	slices.SortStableFunc(res, func(a, b result) bool { return a.distance < b.distance })

	if len(tips) > 1 {
		answers, others, _ := calcMultipleChoice(ctx, choices, tips)
		if len(answers) > 1 && len(others) > 0 {
			log.Print("选择未出现内容")
			switch len(others) {
			case 1:
				return others[0]
			default:
				var res []result
				for _, answer := range answers {
					str = strings.Replace(str, answer, "", 1)
				}
				for _, other := range others {
					res = append(res, result{other, diff.DiffLevenshtein(diff.DiffMain(str, other, false))})
				}
				slices.SortStableFunc(res, func(a, b result) bool { return a.distance > b.distance })

				return res[0].text
			}
		}
	}

	return res[0].text
}

func calcMultipleChoice(ctx context.Context, choices, tips []string) (answers, others []string, incalculable bool) {
	var selected []int
	for i, choice := range choices {
		if slices.Contains(tips, choice) {
			answers = append(answers, choice)
			selected = append(selected, i)
		}
	}

	str := strings.Join(tips, "")
	for _, i := range answers {
		str = strings.Replace(str, i, "", 1)
	}

	done := make(chan struct{})
	go func() {
		for {
			if str == "" {
				close(done)
				return
			}
			for i, choice := range choices {
				if !slices.Contains(selected, i) && strings.HasPrefix(str, choice) {
					answers = append(answers, choice)
					selected = append(selected, i)
					str = strings.Replace(str, choice, "", 1)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		incalculable = true
	case <-done:
	}

	for i, choice := range choices {
		if !slices.Contains(selected, i) {
			others = append(others, choice)
		}
	}

	return
}

func convertNodes(nodes []*cdp.Node, n int) (res []string) {
	for i, node := range nodes {
		if i%n == n-1 {
			res = append(res, node.NodeValue)
		}
	}
	return
}
