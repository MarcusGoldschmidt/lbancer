package pkg

import (
	"errors"
	"github.com/rs/cors"
	"log"
	"net/http"
)

func (lb *LoadBalancer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend, err := lb.Next()
		if err != nil {
			if errors.Is(err, ErrNoServiceAvailable) {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		backend.ServeHTTP(w, r)
	})

	handler := cors.AllowAll().Handler(baseHandler)
	handler = setupLogs(handler)

	handler.ServeHTTP(writer, request)
}

func (lb *LoadBalancer) setRoutesAndServer() {

	lb.server = &http.Server{
		Addr:    ":8080",
		Handler: lb,
	}
}

func setupLogs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &wrappedWriter{w, http.StatusOK}

		next.ServeHTTP(ww, r)

		log.Printf("%s %s %d", r.Method, r.URL.Path, ww.statusCode)
	})
}
