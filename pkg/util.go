package pkg

func MakeTestServerAndBackend() (*TestServer, *Backend, error) {
	server, err := NewTestServer()
	if err != nil {
		return nil, nil, err
	}
	backend, err := NewBackendUrl(server.Addr())
	if err != nil {
		return nil, nil, err
	}
	return server, backend, nil
}
