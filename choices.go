package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slices"
)

var (
	trueOrFalseChoices = [][]string{{"正确", "错误"}, {"错误", "正确"}}
	trueOrFalseAnswer  = map[bool]string{true: "正确", false: "错误"}
	negativeWords      = regexp.MustCompile(`不|无|没|非|免`)
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

	choicesList, tipsList := convertNodes(choices, 3), convertNodes(tips, 1)
	for _, i := range trueOrFalseChoices {
		if reflect.DeepEqual(choicesList, i) {
			log.Print("是非题")
			answers = []string{calcTrueOrFalse(body, strings.Join(tipsList, ""))}
			return
		}
	}

	n := strings.Count(body, "（）")
	switch header[:9] {
	case "单选题":
		log.Print("单选题")
		answers = []string{calcSingleChoice(choicesList, tipsList)}
		return
	case "多选题":
		log.Printf("多选题(%d)", n)
	default:
		log.Printf("未知题型: %s(%d)", header, n)
	}

	if n == len(choicesList) {
		for _, choice := range choicesList {
			answers = append(answers, choice)
		}
		return
	}

	answers, _, incalculable = calcMultipleChoice(choicesList, tipsList)
	if incalculable {
		log.Print("无法计算答案")
		log.Println("题目:", body)
		printTips(tips)
		printChoices(choices)
	}

	return
}

func calcTrueOrFalse(body, tip string) string {
	if slices.Contains(trueOrFalseChoices, tip) {
		return tip
	}

	wb, wt := negativeWords.FindAllString(body, -1), negativeWords.FindAllString(tip, -1)
	return trueOrFalseAnswer[(len(wb)-len(wt))%2 == 0]
}

var diff = diffmatchpatch.New()

func calcSingleChoice(choices, tips []string) string {
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
		answers, others, _ := calcMultipleChoice(choices, tips)
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

func calcMultipleChoice(choices, tips []string) (answers, others []string, incalculable bool) {
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

	for i, choice := range choices {
		if !slices.Contains(selected, i) && strings.Contains(str, choice) {
			answers = append(answers, choice)
			selected = append(selected, i)
			str = strings.Replace(str, choice, "", 1)
		}
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
