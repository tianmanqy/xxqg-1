package main

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestCalcSingleChoice(t *testing.T) {
	testcase := []struct {
		choices []string
		tips    []string
		res     string
	}{
		{
			[]string{"线上", "线下", "线上线下同步"},
			[]string{"线上线下同步"},
			"线上线下同步",
		},
		{
			[]string{"正确", "错误"},
			[]string{"线上线下同步"},
			"正确",
		},
		{
			[]string{"正确说", "错误说"},
			[]string{"正确"},
			"正确说",
		},
		{
			[]string{"大城市 农村", "农村 大城市"},
			[]string{"大城市", "农村"},
			"大城市 农村",
		},
		{
			[]string{"大城市 农村", "农村 大城市"},
			[]string{"大", "城", "市", "农村"},
			"大城市 农村",
		},
		{
			[]string{"A", "B", "C", "D"},
			[]string{"A", "B", "D"},
			"C",
		},
		{
			[]string{"A", "B", "C", "D"},
			[]string{"AA", "B", "C"},
			"D",
		},
	}

	for _, tc := range testcase {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if res := calcSingleChoice(ctx, tc.choices, tc.tips); !reflect.DeepEqual(tc.res, res) {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
		cancel()
	}
}

func TestCalcMultipleChoice(t *testing.T) {
	testcase := []struct {
		choices []string
		tips    []string
		res     []string
	}{
		{
			[]string{"大城市", "农村"},
			[]string{"大城市", "农村"},
			[]string{"大城市", "农村"},
		},
		{
			[]string{"大城市", "农村"},
			[]string{"大", "城", "市", "农村"},
			[]string{"农村", "大城市"},
		},
	}

	for _, tc := range testcase {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if res, _, _ := calcMultipleChoice(ctx, tc.choices, tc.tips); !reflect.DeepEqual(tc.res, res) {
			t.Errorf("expected %q; got %q", tc.res, res)
		}
		cancel()
	}
}
