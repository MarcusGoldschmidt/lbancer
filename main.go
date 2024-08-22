package main

import (
	"lblancer/pkg"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	lb := pkg.NewLoadBalancer()

	go func() {
		err := lb.Start()
		if err != nil {
			log.Println(err)
		}
	}()

	defer lb.Stop()

	err := MakeMultiplesServers(lb, 5)
	if err != nil {
		log.Println(err)
		return
	}

	// Control how many connections are open
	for i := 0; i < 500; i++ {
		go MakeRequestLoop(lb)
	}

	for {
		log.Println("COUNT CONNECTIONS", lb.CountConnections())

		time.Sleep(1 * time.Second)
	}
}

func MakeRequestLoop(lb *pkg.LoadBalancer) {
	for {
		req, err := http.NewRequest("GET", "/hello", nil)
		if err != nil {
			log.Println(err)
			continue
		}

		rr := httptest.NewRecorder()

		lb.ServeHTTP(rr, req)

		log.Println("Response", rr.Code)

		time.Sleep(200 * time.Millisecond)
	}
}

func MakeMultiplesServers(lb *pkg.LoadBalancer, count int) error {
	for i := 0; i < count; i++ {
		s, b, err := pkg.MakeTestServerAndBackend()
		if err != nil {
			return err
		}
		lb.Add(b)
		MakeFail(s)
	}

	return nil
}

func MakeFail(server *pkg.TestServer) {
	go func() {
		for {
			if rand.Float32() < 0.9 {
				server.SetHealthStatus(http.StatusOK)
			} else {
				server.SetHealthStatus(http.StatusInternalServerError)
			}

			time.Sleep(2 * time.Second)
		}
	}()
}
