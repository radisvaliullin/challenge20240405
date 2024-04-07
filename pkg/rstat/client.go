package rstat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ClientConfig struct {
	Token     string
	Subreddit string
}

type Client struct {
	config ClientConfig

	cln *http.Client
}

func NewClient(config ClientConfig) *Client {
	cln := &Client{
		config: config,
	}
	cln.cln = &http.Client{
		Timeout: time.Second * 5,
	}
	return cln
}

func (c *Client) SubredditNew(after, before string) (Resp, error) {

	resp := Resp{}

	// config
	limit := "100"

	vals := url.Values{}
	if len(after) != 0 {
		vals.Set("after", after)
	}
	if len(before) != 0 {
		vals.Set("before", before)
	}
	if len(limit) != 0 {
		vals.Set("limit", limit)
	}

	reqURL := url.URL{
		Scheme:   "https",
		Host:     "oauth.reddit.com",
		Path:     fmt.Sprintf("r/%s/new", c.config.Subreddit),
		RawQuery: vals.Encode(),
	}
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		log.Fatalf("client: new request build error: %v", err)
	}
	req.Header.Set("User-Agent", "ChangeMeClient/0.1 by YourUsername")
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.config.Token))

	// request
	httpRes, err := c.cln.Do(req)
	if err != nil {
		log.Printf("client: request error, %v", err)
		return Resp{}, err
	}
	defer httpRes.Body.Close()

	// handle header
	resp.Header = Header{
		Used:      httpRes.Header.Get("x-ratelimit-used"),
		Remaining: httpRes.Header.Get("x-ratelimit-remaining"),
		Reset:     httpRes.Header.Get("x-ratelimit-reset"),
	}

	// error response
	if httpRes.StatusCode != 200 {
		errBody, err := io.ReadAll(httpRes.Body)
		if err != nil {
			log.Printf("client: response body read error, %v", err)
			return resp, err
		}
		if len(errBody) > 256 {
			errBody = errBody[:256]
		}
		err = fmt.Errorf("client: error response, code - %v, message - %v", httpRes.StatusCode, string(errBody))
		return resp, err
	}

	err = json.NewDecoder(httpRes.Body).Decode(&resp.Payload)
	if err != nil {
		log.Printf("client: response decode error: %v", err)
		return resp, err
	}

	return resp, nil
}
