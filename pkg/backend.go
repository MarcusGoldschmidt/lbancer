package pkg

import (
	"github.com/google/uuid"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type BackendStatus string

const (
	// initial, healthy, unhealthy, closed
	BackendStatusInitial   BackendStatus = "initial"
	BackendStatusHealthy   BackendStatus = "healthy"
	BackendStatusUnhealthy BackendStatus = "unhealthy"
	BackendStatusClosed    BackendStatus = "closed"
)

type HeartBeatChecker interface {
	IsHealthy() bool

	Interval() time.Duration
}

type Backend struct {
	id           uuid.UUID
	url          *url.URL
	mux          *sync.RWMutex
	reverseProxy *httputil.ReverseProxy

	status      BackendStatus
	connections int32

	cancel        chan struct{}
	heartBeatOnce sync.Once

	heartBeatChecker HeartBeatChecker

	wg *sync.WaitGroup
}

func (b *Backend) Id() uuid.UUID {
	return b.id
}

func (b *Backend) Url() *url.URL {
	return b.url
}

func (b *Backend) Status() BackendStatus {
	return b.status
}

func (b *Backend) Connections() int32 {
	return b.connections
}

func (b *Backend) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	b.wg.Add(1)
	atomic.AddInt32(&b.connections, 1)
	b.reverseProxy.ServeHTTP(writer, request)
	atomic.AddInt32(&b.connections, -1)
}

func NewBackendUrl(backendUrl string, heartBeatOption ...HeartBeatChecker) (*Backend, error) {
	u, err := url.ParseRequestURI(backendUrl)
	if err != nil {
		return nil, err
	}

	return NewBackend(*u, heartBeatOption...), nil
}

func NewBackend(backendUrl url.URL, heartBeatOption ...HeartBeatChecker) *Backend {
	var heartBearChecker HeartBeatChecker

	if len(heartBeatOption) > 0 {
		heartBearChecker = heartBeatOption[0]
	} else {
		u := backendUrl
		u.Path = "/health"
		u.RawPath = "/health"

		heartBearChecker = NewBasicHeartBeat(u, 5*time.Second)
	}

	return &Backend{
		id:            uuid.New(),
		url:           &backendUrl,
		heartBeatOnce: sync.Once{},
		status:        BackendStatusInitial,
		cancel:        make(chan struct{}),
		mux:           &sync.RWMutex{},
		connections:   0,
		reverseProxy: &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = backendUrl.Scheme
				req.URL.Host = backendUrl.Host
				req.URL.Path = backendUrl.Path
				req.URL.RawPath = backendUrl.RawPath
				req.URL.RawQuery = backendUrl.RawQuery
			},
		},
		heartBeatChecker: heartBearChecker,
		wg:               &sync.WaitGroup{},
	}
}

func (b *Backend) Cancel() {
	b.mux.Lock()
	defer b.mux.Unlock()

	if b.status == BackendStatusClosed {
		return
	}

	b.status = BackendStatusClosed

	close(b.cancel)

	b.wg.Wait()
}

func (b *Backend) StartHeartBeat() {
	b.heartBeatOnce.Do(func() {
		go func() {

			ticker := time.NewTicker(b.heartBeatChecker.Interval())

			for {
				select {
				case <-b.cancel:
					return
				case <-ticker.C:
					b.CheckHealthy()
				}
			}
		}()
	})
}

func (b *Backend) CheckHealthy() {
	b.mux.Lock()
	defer b.mux.Unlock()

	if b.heartBeatChecker.IsHealthy() {
		log.Printf("Backend %s is healthy", b.Url().String())
		b.status = BackendStatusHealthy
	} else {
		log.Printf("Backend %s is unhealthy", b.Url().String())
		b.status = BackendStatusUnhealthy
	}
}
