package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/sunshineplan/chrome"
)

const (
	paperAPI = "https://pc-proxy-api.xuexi.cn/api/exam/service/paper/pc/list"
	scoreAPI = "https://pc-proxy-api.xuexi.cn/api/exam/service/detail/score"

	examLimit = 15 * time.Second

	captchaAPI = "https://cf.aliyun.com/nocaptcha/analyze.jsonp"
)

var examStatus = true

func exam(ctx context.Context, url, class string) (err error) {
	if !examStatus {
		log.Print("检测到验证滑块未通过，跳过答题")
		return
	}

	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	if err = chromedp.Run(ctx); err != nil {
		return
	}

	navCtx, cancel := context.WithTimeout(ctx, examLimit)
	defer cancel()

	if err = chromedp.Run(navCtx, chromedp.Navigate(url)); err != nil {
		return
	}

	var title string
	if class == "" {
		title = "每日答题"
	} else {
		var page int
		var buttons []*cdp.Node
		page, err = getPageNumber(navCtx)
		if err != nil {
			return err
		}

		pageCtx, cancel := context.WithTimeout(ctx, time.Duration(page)*time.Second)
		defer cancel()

		more := chrome.ListenEvent(pageCtx, paperAPI, "GET", false)
		for i := 0; i < page; i++ {
			if err = chromedp.Run(
				pageCtx,
				chromedp.WaitVisible("div.ant-spin-container"),
				chromedp.Nodes(fmt.Sprintf("div.%s button:not(.ant-btn-background-ghost)", class), &buttons, chromedp.AtLeast(0)),
			); err != nil {
				return
			}

			if len(buttons) != 0 || i == page-1 {
				break
			} else {
				time.Sleep(200 * time.Millisecond)
				if err = chromedp.Run(pageCtx, chromedp.Click(`li[title="Next Page"][aria-disabled=false]`)); err != nil {
					return
				}

				select {
				case <-pageCtx.Done():
					return pageCtx.Err()
				case <-more:
				}
			}
		}

		if len(buttons) == 0 {
			return fmt.Errorf("没有可用试题")
		}

		if err = chromedp.Run(
			pageCtx,
			chromedp.MouseClickNode(buttons[0]),
			chromedp.WaitVisible("div.question"),
			chromedp.Text("div.title", &title, chromedp.AtLeast(0)),
		); err != nil {
			return
		}
	}

	n, err := getExamNumber(ctx)
	if err != nil {
		return
	}
	log.Printf("[答题] %s(共%d题)", title, n)

	ctx, cancel = context.WithTimeout(ctx, time.Duration(n)*examLimit)
	defer cancel()

	start := time.Now()
	done := chrome.ListenEvent(ctx, scoreAPI, "GET", false)
	go func() {
		for i := 1; i <= n; i++ {
			log.Printf("#题目%d", i)
			var tips, inputs, choices []*cdp.Node
			var body, tip string
			if err = chromedp.Run(
				ctx,
				chromedp.Click("span.tips", chromedp.NodeVisible),
				chromedp.Text("div.line-feed", &tip),
				chromedp.Nodes(`//div[@class="line-feed"]//font[@color="red"]/text()`, &tips, chromedp.AtLeast(0)),
				chromedp.Click("div.q-header>svg"),
				chromedp.WaitNotVisible("div.line-feed"),
				chromedp.Nodes("input.blank", &inputs, chromedp.AtLeast(0)),
				chromedp.Text("div.q-body", &body),
			); err != nil {
				return
			}
			if len(tips) == 0 {
				log.Print("没有提示答案")
			}

			var answers []string
			var incalculable bool
			if len(inputs) == 0 {
				choices, answers, incalculable, err = getChoiceQuestionAnswers(ctx, body, tip, tips)
				if err != nil {
					return
				}

				if len(answers) != 0 {
					for _, i := range answers {
						log.Println("选择", i)
						if err = chromedp.Run(
							ctx,
							chromedp.Sleep(time.Second),
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
							str := randomString(body, rand.Intn(3)+2)
							log.Println("随机输入", str)
							if err = chromedp.Run(ctx, chromedp.KeyEventNode(input, str)); err != nil {
								return
							}
						}
					}
				}
			}

			if class != paperClass || i < n {
				if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.next-btn", chromedp.NodeEnabled)); err != nil {
					return
				}

				var nodes []*cdp.Node
				if err = chromedp.Run(
					ctx,
					chromedp.Sleep(time.Second),
					chromedp.Nodes("div.action-row>button.next-btn:enabled", &nodes, chromedp.AtLeast(0)),
				); err != nil {
					return
				}

				if len(nodes) != 0 {
					var answer string
					if err := chromedp.Run(
						ctx,
						chromedp.EvaluateAsDevTools(`$("div.answer").innerText`, &answer),
						chromedp.Click("div.action-row>button.next-btn"),
					); err == nil {
						log.Print("答错 ×")
						if len(inputs) == 0 && !incalculable {
							log.Println("题目:", body)
							log.Println("提示:", tip)
							printTips(tips)
							printChoices(choices)
						}
						log.Print(answer)
					} else {
						if strings.Contains(err.Error(), "Cannot read properties of null (reading 'innerText')") {
							log.Print("答对 √")
						} else {
							log.Println("无法获取答案:", err)
							if len(inputs) == 0 && !incalculable {
								log.Println("题目:", body)
								log.Println("提示:", tip)
								printTips(tips)
								printChoices(choices)
							}
						}
					}
				} else {
					if class != paperClass {
						log.Print("答对 √")
					}
				}
			}

			if i == n && class == paperClass {
				if err = chromedp.Run(ctx, chromedp.Click("div.action-row>button.submit-btn", chromedp.NodeEnabled)); err != nil {
					return
				}
			}

			if err = checkCaptcha(ctx); err != nil {
				examStatus = false
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	log.Println("答题完毕！耗时:", time.Since(start))
	return
}

func getPageNumber(ctx context.Context) (n int, err error) {
	var buttons []*cdp.Node
	if err = chromedp.Run(ctx, chromedp.Nodes("li.ant-pagination-item", &buttons)); err != nil {
		return
	}

	if length := len(buttons); length == 0 {
		err = fmt.Errorf("no pagination item found")
	} else {
		n, err = strconv.Atoi(buttons[length-1].AttributeValue("title"))
	}

	return
}

func getExamNumber(ctx context.Context) (n int, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var pager string
	if err = chromedp.Run(
		ctx,
		chromedp.WaitVisible("div.question"),
		chromedp.Text("div.pager", &pager),
	); err != nil {
		return
	}

	if res := regexp.MustCompile(`\d+/(\d+)`).FindStringSubmatch(pager); len(res) == 2 {
		n, err = strconv.Atoi(res[1])
	} else {
		err = fmt.Errorf("获取题目数量失败: %s", pager)
	}

	return
}

func checkCaptcha(ctx context.Context) error {
	check, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	if err := chromedp.Run(check, chromedp.WaitVisible("div#nc_mask")); err != nil {
		return nil
	}
	log.Print("出现验证滑块")

	slide, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	x, y, err := coordinate(slide, "span.btn_slide")
	if err != nil {
		return err
	}

	done := chrome.ListenEvent(slide, captchaAPI, "GET", true)
	if err := chromedp.Run(
		slide,
		chromedp.MouseEvent(input.MousePressed, x, y, chromedp.ButtonLeft, chromedp.ClickCount(1)),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+43, y),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+86, y),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+129, y),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+172, y),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+215, y),
		chromedp.Sleep(time.Millisecond*150),
		chromedp.MouseEvent(input.MouseMoved, x+258, y),
		chromedp.MouseEvent(input.MouseReleased, x+258, y, chromedp.ButtonLeft),
	); err != nil {
		return err
	}

	select {
	case <-slide.Done():
		return slide.Err()
	case e := <-done:
		if s := string(e.Bytes); strings.Contains(s, `"success":true`) {
			return nil
		} else {
			log.Print(s)
			return fmt.Errorf("验证滑块未通过")
		}
	}
}

func coordinate(ctx context.Context, sel any) (float64, float64, error) {
	var model *dom.BoxModel
	if err := chromedp.Run(ctx, chromedp.Dimensions(sel, &model)); err != nil {
		return 0, 0, err
	}
	return model.Border[0] + float64(model.Width)/2, model.Border[1] + float64(model.Height)/2, nil
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
