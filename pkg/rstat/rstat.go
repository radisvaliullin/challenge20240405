package rstat

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func Main() {

	conf, err := GetConfig()
	if err != nil {
		log.Fatalf("rstat: get config error, %v", err)
	}
	cln := NewClient(conf.Client)
	clnLimiter := NewClientLimiter(cln)
	rstat := New(conf, clnLimiter)

	if err = rstat.Run(); err != nil {
		log.Printf("rstat: run error, %v", err)
	}
}

type RStat struct {
	config Config

	cln IClient

	// just for testing
	// atomic
	numSimultFetch int64

	//
	mx        sync.Mutex
	lastStart float64
}

func New(config Config, cln IClient) *RStat {
	s := &RStat{
		config: config,
		cln:    cln,
	}
	return s
}

func (s *RStat) Run() error {
	fmt.Printf("subreddit %s, new posts:\n", s.config.Client.Subreddit)

	// init lastStart
	res, err := s.cln.SubredditNew("", "")
	if err != nil {
		log.Printf("rstat: request error, %v", err)
		return err
	}
	s.mx.Lock()
	if len(res.Payload.Data.Children) > 0 {
		s.lastStart = res.Payload.Data.Children[0].Data.Created
	}
	s.mx.Unlock()

	statTick := time.NewTicker(time.Second * 10)
	for {
		select {
		default:
			s.fetchNewPosts()
		case <-statTick.C:
			fmt.Println("stat: total requests - ", s.cln.GetTotalReqCnt())
			fmt.Println("stat: num simult - ", atomic.LoadInt64(&s.numSimultFetch))
		}
	}
}

func (s *RStat) fetchNewPosts() {
	atomic.AddInt64(&s.numSimultFetch, 1)
	defer func() {
		atomic.AddInt64(&s.numSimultFetch, -1)
	}()

	res, err := s.cln.SubredditNew("", "")
	if err != nil {
		log.Printf("rstat: fetch new posts error, %v", err)
		return
	}

	// no data
	if len(res.Payload.Data.Children) == 0 {
		return
	}

	// get time range and update lastStart
	var start float64
	var end float64
	// fetch posts time range [start, end)
	start = res.Payload.Data.Children[0].Data.Created
	// update lastStart time
	s.mx.Lock()
	if start > s.lastStart {
		end = s.lastStart
		s.lastStart = start
	} else {
		// no new data, return
		s.mx.Unlock()
		return
	}
	s.mx.Unlock()

	go s.fetchPostsForRange(start, end, res)
}

func (s *RStat) fetchPostsForRange(start, end float64, firstPage Resp) {
	atomic.AddInt64(&s.numSimultFetch, 1)
	defer func() {
		atomic.AddInt64(&s.numSimultFetch, -1)
	}()

	res := firstPage
	for {
		for _, child := range res.Payload.Data.Children {
			if child.Data.Created > start {
				continue
			}
			if child.Data.Created <= end {
				// we handled all items for timerange [start, end)
				return
			}
			fmt.Printf("new post: %+v", child)
		}

		// handle next page
		after := res.Payload.Data.After
		if len(after) == 0 {
			return
		}

		//
		for {
			var err error
			res, err = s.cln.SubredditNew(after, "")
			if err != nil {
				log.Printf("rstat: fetch posts error, %v", err)
				continue
			}
			break
		}

		// no data
		if len(res.Payload.Data.Children) == 0 {
			return
		}
	}
}
