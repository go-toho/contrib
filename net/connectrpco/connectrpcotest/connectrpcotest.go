// Package connectrpcotest provides utilities for testing Connect services.
package connectrpcotest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/go-toho/contrib/net/connectrpco"
)

// Server is a Connect server listening on a local loopback address.
type Server struct {
	*httptest.Server

	httpClient *http.Client
}

// NewServer starts a Connect server configured with the same options as
// connectrpco.NewConnectServer, so tests exercise the production wiring. The
// server is closed when the test ends.
func NewServer(t testing.TB, opts ...connectrpco.ConnectServerOption) *Server {
	t.Helper()

	srv, err := connectrpco.NewConnectServer(opts...)
	if err != nil {
		t.Fatalf("could not create Connect server: %v", err)
	}

	// HTTP/2 without TLS supports every RPC type, including bidirectional
	// streaming
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	transport := &http.Transport{Protocols: protocols}

	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(func() {
		transport.CloseIdleConnections()
		ts.Close()
	})

	return &Server{
		Server:     ts,
		httpClient: &http.Client{Transport: transport},
	}
}

// HttpClient returns an HTTP client for the server that speaks HTTP/2 without
// TLS.
func (s *Server) HttpClient() *http.Client {
	return s.httpClient
}

// Transport returns a Connect transport for the server.
func (s *Server) Transport(opts ...connecthttp.Option) connect.Transport {
	return connecthttp.NewTransport(s.HttpClient(), s.URL, opts...)
}

// Client returns a Connect client for the server, to pass to generated
// New<Service>Client constructors.
func (s *Server) Client(interceptors ...connect.ClientInterceptor) *connect.Client {
	return connect.NewClient(s.Transport(), interceptors...)
}
