package pkg

import (
	"context"
	"github.com/rs/cors"
	"log"
	"net"
	"net/http"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type TestServer struct {
	server *http.Server

	healthStatus int

	err error
}

func (s *TestServer) HealthStatus() int {
	return s.healthStatus
}

func (s *TestServer) SetHealthStatus(healthStatus int) {
	s.healthStatus = healthStatus
}

func NewTestServer() (*TestServer, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	server := &TestServer{
		healthStatus: 200,
	}

	router := http.NewServeMux()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(server.healthStatus)
	})

	handler := cors.AllowAll().Handler(router)
	handler = setupLogs(handler)

	server.server = &http.Server{
		Addr:    "localhost" + listener.Addr().String()[4:],
		Handler: handler,
	}

	go func() {
		err := server.server.Serve(listener)
		if err != nil {
			server.err = err
			log.Println(err)
		}
	}()

	server.server.Addr = listener.Addr().String()
	return server, nil
}

func (s *TestServer) Addr() string {
	host, port, err := net.SplitHostPort(s.server.Addr)
	if err != nil {
		return s.server.Addr
	}

	if host == "::" {
		host = "localhost"
	}

	return "http://" + host + ":" + port
}

func (t *TestServer) Shutdown(ctx context.Context) error {
	return t.server.Shutdown(ctx)
}
