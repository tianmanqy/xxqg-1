package main

import (
	"sync"
	"sync/atomic"
)

type status struct {
	sync.Mutex
	article, video   atomic.Int64
	run, pause, done chan struct{}
}

func newStatus(article, video int64) *status {
	s := &status{
		run:  make(chan struct{}),
		done: make(chan struct{}),
	}
	s.article.Store(article)
	s.video.Store(video)
	s.Run()
	return s
}

func (s *status) Run() {
	s.Lock()
	defer func() {
		recover()
		s.Unlock()
	}()
	s.pause = make(chan struct{})
	close(s.run)
}

func (s *status) Pause() {
	s.Lock()
	defer func() {
		recover()
		s.Unlock()
	}()
	s.run = make(chan struct{})
	close(s.pause)
}

func (s *status) Done() {
	close(s.done)
}

func (s *status) addArticle(n int64) {
	if n := s.article.Add(n); n > 0 {
		s.Run()
	}
}

func (s *status) reduceArticle() {
	s.article.Add(-1)
}

func (s *status) addVideo(n int64) {
	if n := s.video.Add(n); n > 0 {
		s.Run()
	}
}

func (s *status) reduceVideo() {
	s.video.Add(-1)
}
