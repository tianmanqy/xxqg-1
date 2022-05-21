package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
)

func TestCalcSingleChoice(t *testing.T) {
	testcase := []struct {
		choices []*cdp.Node
		tip     string
		res     string
	}{
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "线上"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "线下"},
				{NodeValue: "C"},
				{NodeValue: "."},
				{NodeValue: "线上线下同步"},
			},
			"线上线下同步",
			"线上线下同步",
		},
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "正确"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "错误"},
			},
			"线上线下同步",
			"A",
		},
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "正确说"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "错误说"},
			},
			"正确",
			"正确说",
		},
	}

	for _, tc := range testcase {
		if res := calcSingleChoice(tc.choices, tc.tip); !reflect.DeepEqual(tc.res, res) {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
	}
}

func TestCalcMultipleChoice(t *testing.T) {
	testcase := []struct {
		choices []*cdp.Node
		tips    []*cdp.Node
		res     []string
	}{
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "大城市"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "农村"},
			},
			[]*cdp.Node{{NodeValue: "大城市"}, {NodeValue: "农村"}},
			[]string{"大城市", "农村"},
		},
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "大城市"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "农村"},
			},
			[]*cdp.Node{{NodeValue: "大"}, {NodeValue: "城"}, {NodeValue: "市"}, {NodeValue: "农村"}},
			[]string{"大城市", "农村"},
		},
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "大城市 农村"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "农村 大城市"},
			},
			[]*cdp.Node{{NodeValue: "大城市"}, {NodeValue: "农村"}},
			[]string{"大城市 农村"},
		},
		{
			[]*cdp.Node{
				{NodeValue: "A"},
				{NodeValue: "."},
				{NodeValue: "大城市 农村"},
				{NodeValue: "B"},
				{NodeValue: "."},
				{NodeValue: "农村 大城市"},
			},
			[]*cdp.Node{{NodeValue: "大"}, {NodeValue: "城"}, {NodeValue: "市"}, {NodeValue: "农村"}},
			[]string{"大城市 农村"},
		},
	}

	for _, tc := range testcase {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if res := calcMultipleChoice(ctx, tc.choices, tc.tips); !reflect.DeepEqual(tc.res, res) {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
		cancel()
	}
}
