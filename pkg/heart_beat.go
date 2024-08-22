package pkg

import (
	"net/http"
	"net/url"
	"time"
)

type BasicHeartBeat struct {
	url      url.URL
	interval time.Duration
}

func NewBasicHeartBeat(url url.URL, interval time.Duration) *BasicHeartBeat {
	return &BasicHeartBeat{
		url:      url,
		interval: interval,
	}
}

func (b *BasicHeartBeat) IsHealthy() bool {
	client := &http.Client{}

	response, err := client.Get(b.url.String())
	if err != nil {
		return false
	}

	response.Body.Close()

	return response.StatusCode == http.StatusOK
}

func (b *BasicHeartBeat) Interval() time.Duration {
	return b.interval
}
