package rstat

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func Main() {

	conf, err := GetConfig()
	if err != nil {
		log.Fatalf("rstat: get config error, %v", err)
	}
	cln := NewClient(conf.Client)
	rstat := New(conf, cln)

	if err = rstat.Run(); err != nil {
		log.Printf("rstat: run error, %v", err)
	}
}

type RStat struct {
	config Config

	cln *Client

	//
	mx          sync.Mutex
	totalReqCnt int
	lastHandle  float64
}

func New(config Config, cln *Client) *RStat {
	s := &RStat{
		config: config,
		cln:    cln,
	}
	return s
}

func (s *RStat) Run() error {
	fmt.Printf("subreddit %s, new posts:\n", s.config.Client.Subreddit)

	// init
	res, err := s.cln.SubredditNew("", "")
	if err != nil {
		log.Printf("rstat: request error: %v", err)
	}
	s.mx.Lock()
	if len(res.Payload.Data.Children) > 0 {
		s.lastHandle = res.Payload.Data.Children[0].Data.Created
	}
	s.mx.Unlock()

	reqTickDur := s.config.PereodicReqDur
	reqTick := time.NewTicker(reqTickDur)
	statTick := time.NewTicker(time.Second * 10)

	for {
		select {
		case <-reqTick.C:

			go s.updatePosts()

		case <-statTick.C:
			func() {
				s.mx.Lock()
				defer s.mx.Unlock()
				fmt.Println("stat:", s.totalReqCnt)
			}()
		}
	}
}

func (s *RStat) updatePosts() {
	res, err := s.cln.SubredditNew("", "")
	if err != nil {
		log.Printf("rstat: request error: %v", err)
		return
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	s.totalReqCnt++
	if len(res.Payload.Data.Children) == 0 {
		return
	}

	var start = res.Payload.Data.Children[0].Data.Created
	var end float64
	if start > s.lastHandle {
		end = s.lastHandle
		s.lastHandle = start
	} else {
		// no new data, return
		return
	}

	newItemCnt := 0
	for _, child := range res.Payload.Data.Children {
		if child.Data.Created <= end {
			break
		}
		newItemCnt++
		fmt.Printf("new post: %+v", child)
	}
	if newItemCnt == 100 {
		fmt.Println("warning: you missed handle some of posts")
	}
}
