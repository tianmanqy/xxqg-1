package main

import (
	"reflect"
	"testing"
)

func TestCalcTrueOrFalse(t *testing.T) {
	testcase := []struct {
		body string
		tip  string
		res  string
	}{
		{
			"线上线下同步",
			"线上线下同步",
			"正确",
		},
		{
			"是",
			"不是",
			"错误",
		},
		{
			"不同",
			"并非不同",
			"错误",
		},
	}

	for _, tc := range testcase {
		if res := calcTrueOrFalse(tc.body, tc.tip); tc.res != res {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
	}
}

func TestCalcSingleChoice(t *testing.T) {
	testcase := []struct {
		number  int
		tip     string
		choices []string
		tips    []string
		res     string
	}{
		{
			1,
			"",
			[]string{"线上", "线下", "线上线下同步"},
			[]string{"线上线下同步"},
			"线上线下同步",
		},
		{
			1,
			"",
			[]string{"正确说", "错误说"},
			[]string{"正确"},
			"正确说",
		},
		{
			2,
			"",
			[]string{"大城市 农村", "农村 大城市"},
			[]string{"大城市", "农村"},
			"大城市 农村",
		},
		{
			2,
			"",
			[]string{"大城市 农村", "农村 大城市"},
			[]string{"大", "城", "市", "农村"},
			"大城市 农村",
		},
		{
			1,
			"",
			[]string{"A", "B", "C", "D"},
			[]string{"A", "B", "D"},
			"C",
		},
		{
			1,
			"",
			[]string{"A", "B", "C", "D"},
			[]string{"AA", "B", "C"},
			"D",
		},
		{
			1,
			"TATBT",
			[]string{"SAAS", "SABS", "SACS", "SADS"},
			[]string{"A", "B"},
			"SADS",
		},
		{
			1,
			"TABT",
			[]string{"SAAS", "SABS", "SACS", "SADS"},
			[]string{"A", "B"},
			"SABS",
		},
		{
			1,
			"A,B,C,D",
			[]string{"A-B-E-F", "A-B-F-C", "A-B-C-D"},
			[]string{"A", "B", "C", "D"},
			"A-B-C-D",
		},
	}

	for _, tc := range testcase {
		if res := calcSingleChoice(tc.number, tc.tip, tc.choices, tc.tips); tc.res != res {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
	}
}

func TestCalcMultipleChoice(t *testing.T) {
	testcase := []struct {
		number  int
		choices []string
		tips    []string
		res     []string
	}{
		{
			2,
			[]string{"大城市", "农村"},
			[]string{"大城市", "农村"},
			[]string{"大城市", "农村"},
		},
		{
			2,
			[]string{"大城市", "农村"},
			[]string{"大", "城", "市", "农村"},
			[]string{"农村", "大城市"},
		},
		{
			3,
			[]string{"A", "B", "C", "D"},
			[]string{"BB", "C", "D", "E"},
			[]string{"C", "D", "B"},
		},
	}

	for _, tc := range testcase {
		if res, _, _ := calcMultipleChoice(tc.number, tc.choices, tc.tips); !reflect.DeepEqual(tc.res, res) {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
	}
}
