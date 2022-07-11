package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/exp/slices"
)

const (
	trueStr  = "正确"
	falseStr = "错误"

	choiceLimit = examLimit / 2
)

var (
	trueOrFalseChoices = [][]string{{trueStr, falseStr}, {falseStr, trueStr}}
	trueOrFalseAnswer  = map[bool]string{true: trueStr, false: falseStr}
	negativeWords      = regexp.MustCompile(`不|无|没|非|免`)
)

func getChoiceQuestionAnswers(ctx context.Context, body, tip string, tips []*cdp.Node) (
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

	n := len(regexp.MustCompile(`[\(（]\s*[\)）]`).FindAllString(body, -1))
	switch header[:9] {
	case "单选题":
		log.Print("单选题")
		answers = []string{calcSingleChoice(n, tip, choicesList, tipsList)}
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

	answers, _, incalculable = calcMultipleChoice(n, choicesList, tipsList)
	if incalculable {
		log.Print("无法计算答案")
		log.Println("题目:", body)
		log.Println("提示:", tip)
		printTips(tips)
		printChoices(choices)
	}

	return
}

func calcTrueOrFalse(body, tip string) string {
	for _, i := range []string{trueStr, falseStr} {
		if strings.Contains(tip, i) {
			return i
		}
	}

	wb, wt := negativeWords.FindAllString(body, -1), negativeWords.FindAllString(tip, -1)
	return trueOrFalseAnswer[(len(wb)-len(wt))%2 == 0]
}

var diff = diffmatchpatch.New()

func calcSingleChoice(n int, tip string, choices, tips []string) string {
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

	if !strings.Contains(tip, str) && len(tips) > n && len(choices) > 2 {
		log.Print("选择未出现内容")
		_, others, _ := calcMultipleChoice(len(choices)-1, choices, tips)
		return others[0]
	}

	return res[0].text
}

func calcMultipleChoice(n int, choices, tips []string) (answers, others []string, incalculable bool) {
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

	if n > 0 && len(answers) < n {
		incalculable = true

		type result struct {
			index    int
			text     string
			distance int
		}
		var res []result
		for i, choice := range choices {
			if !slices.Contains(selected, i) {
				res = append(
					res,
					result{i, choice, diff.DiffLevenshtein(diff.DiffMain(str, choice, false)) +
						utf8.RuneCountInString(choice) -
						utf8.RuneCountInString(str)},
				)
			}
		}
		slices.SortStableFunc(res, func(a, b result) bool { return a.distance < b.distance })

		for i, n := 0, n-len(answers); i < n; i++ {
			answers = append(answers, res[i].text)
			selected = append(selected, res[i].index)
		}
	}

	for i, choice := range choices {
		if !slices.Contains(selected, i) {
			others = append(others, choice)
		}
	}

	return
}

func convertNodes(nodes []*cdp.Node, n int) []string {
	var res []string
	for i, node := range nodes {
		if i%n == n-1 && strings.TrimSpace(node.NodeValue) != "" {
			res = append(res, node.NodeValue)
		}
	}
	return slices.Compact(res)
}
