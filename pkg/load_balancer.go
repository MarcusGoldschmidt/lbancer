package pkg

import (
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
)

var ErrNoServiceRegistered = errors.New("no service registered")
var ErrNoServiceAvailable = errors.New("no service available")

type LoadBalancer struct {
	arr     []*Backend
	pointer int
	mutex   *sync.RWMutex
	server  *http.Server
}

func NewLoadBalancer() *LoadBalancer {
	lb := &LoadBalancer{
		arr:     make([]*Backend, 0),
		mutex:   &sync.RWMutex{},
		pointer: 0,
	}

	lb.setRoutesAndServer()

	return lb
}

func (lb *LoadBalancer) Start() error {
	return lb.server.ListenAndServe()
}

func (lb *LoadBalancer) Stop() error {
	return lb.server.Close()
}

func (lb *LoadBalancer) CountConnections() int32 {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	sum := int32(0)

	for _, b := range lb.arr {
		sum += b.Connections()
	}

	return sum
}

func (lb *LoadBalancer) Add(b *Backend) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.arr = append(lb.arr, b)

	b.CheckHealthy()
	b.StartHeartBeat()
}

func (lb *LoadBalancer) Next() (*Backend, error) {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if len(lb.arr) == 0 {
		return nil, ErrNoServiceRegistered
	}

	p := lb.pointer

	for {
		backend := lb.arr[p]
		p = (p + 1) % len(lb.arr)

		if backend.status == BackendStatusHealthy {
			lb.pointer = p
			return backend, nil
		}

		if p == lb.pointer {
			break
		}
	}

	return nil, ErrNoServiceAvailable
}

func (lb *LoadBalancer) Remove(s *Backend) {
	log.Println("Removing service from load balancer")

	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for i, b := range lb.arr {
		if b.id == s.id {
			s.Cancel()
			lb.arr = append(lb.arr[:i], lb.arr[i+1:]...)
			return
		}
	}
}

func (lb *LoadBalancer) Addr() string {
	host, port, err := net.SplitHostPort(lb.server.Addr)
	if err != nil {
		return lb.server.Addr
	}

	if host == "::" {
		host = "localhost"
	}

	return "http://" + host + ":" + port
}
