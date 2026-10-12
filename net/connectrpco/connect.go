package connectrpco

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/creasty/defaults"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/go-toho/contrib/net/httpo"
)

type ConnectConfig struct {
	Enabled           bool          `default:"true"`
	Addr              string        `default:":8080"`
	ReadHeaderTimeout time.Duration `default:"1s"`
	ReadTimeout       time.Duration `default:"30s"`
	WriteTimeout      time.Duration `default:"30s"`
	IdleTimeout       time.Duration `default:"120s"`
	MaxHeaderBytes    int           `default:"8192"` // 8KiB
}

// ConnectServerOption is an Connect server option.
type ConnectServerOption func(*ConnectServer)

// WithConnectConfig with server config.
func WithConnectConfig(config ConnectConfig) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config = &config
	}
}

// Address with server address.
func Address(addr string) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.Addr = addr
	}
}

// ReadHeaderTimeout with server read header timeout.
func ReadHeaderTimeout(timeout time.Duration) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.ReadHeaderTimeout = timeout
	}
}

// ReadTimeout with server read timeout.
func ReadTimeout(timeout time.Duration) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.ReadTimeout = timeout
	}
}

// WriteTimeout with server write timeout.
func WriteTimeout(timeout time.Duration) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.WriteTimeout = timeout
	}
}

// IdleTimeout with server idle timeout.
func IdleTimeout(timeout time.Duration) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.IdleTimeout = timeout
	}
}

// MaxHeaderBytes with server max header bytes.
func MaxHeaderBytes(bytes int) ConnectServerOption {
	return func(s *ConnectServer) {
		s.config.MaxHeaderBytes = bytes
	}
}

// ServiceRegistrars registers Connect services. Registrars run in the order
// they are given, after all server interceptors are known.
func ServiceRegistrars(registrars ...ServiceRegistrar) ConnectServerOption {
	return func(s *ConnectServer) {
		s.serviceRegistrars = append(s.serviceRegistrars, registrars...)
	}
}

// ServerInterceptors with Connect server interceptors. Interceptors apply to
// every registered service; the first one wraps the outermost call.
func ServerInterceptors(interceptors ...connect.ServerInterceptor) ConnectServerOption {
	return func(s *ConnectServer) {
		s.serverInterceptors = append(s.serverInterceptors, interceptors...)
	}
}

// ConnectHttpOptions with Connect HTTP transport options, such as read limits
// or compression settings. They apply to every registered service.
func ConnectHttpOptions(options ...connecthttp.Option) ConnectServerOption {
	return func(s *ConnectServer) {
		s.connectHttpOptions = append(s.connectHttpOptions, options...)
	}
}

// Handle registers a new plain HTTP handler for a given pattern.
func Handle(pattern string, handler http.Handler) ConnectServerOption {
	return func(s *ConnectServer) {
		s.httpHandlers = append(s.httpHandlers, httpo.HttpPatternHandler{
			Pattern: pattern,
			Handler: handler,
		})
	}
}

// HttpPatternHandlers registers plain HTTP handlers.
func HttpPatternHandlers(patternHandlers ...httpo.HttpPatternHandler) ConnectServerOption {
	return func(s *ConnectServer) {
		s.httpHandlers = append(s.httpHandlers, patternHandlers...)
	}
}

// HttpMiddleware with server http middleware.
func HttpMiddleware(middleware ...httpo.HttpMiddleware) ConnectServerOption {
	return func(s *ConnectServer) {
		s.httpMiddleware = append(s.httpMiddleware, middleware...)
	}
}

// ConnectServer is an HTTP server wrapper.
type ConnectServer struct {
	*http.Server

	config             *ConnectConfig
	serviceRegistrars  []ServiceRegistrar
	serverInterceptors []connect.ServerInterceptor
	connectHttpOptions []connecthttp.Option
	httpHandlers       []httpo.HttpPatternHandler
	httpMiddleware     []httpo.HttpMiddleware

	hostPort  string
	rpcServer *connect.Server
	mux       *http.ServeMux
	handler   http.Handler
}

func NewConnectServer(opts ...ConnectServerOption) (*ConnectServer, error) {
	srv := &ConnectServer{
		config: &ConnectConfig{},
	}

	if err := defaults.Set(srv.config); err != nil {
		return nil, fmt.Errorf("could not set defaults: %w", err)
	}

	// apply config options
	for _, opt := range opts {
		opt(srv)
	}

	addr, err := net.ResolveTCPAddr("tcp", srv.config.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve TCP address: %w", err)
	}

	// get the host and port
	srv.hostPort = addr.String()

	// create the Connect server and register services
	srv.rpcServer = connect.NewServer(srv.serverInterceptors...)
	for _, register := range srv.serviceRegistrars {
		if register == nil {
			continue
		}
		register(srv.rpcServer)
	}

	// create a new mux
	srv.mux = http.NewServeMux()

	// add connect routes
	connecthttp.Mount(srv.mux, srv.rpcServer, srv.connectHttpOptions...)

	// add plain HTTP handlers
	for _, handler := range srv.httpHandlers {
		srv.mux.Handle(handler.Pattern, handler.Handler)
	}

	// extract the handler from the mux
	srv.handler = srv.mux

	// chain handlers (aka "HTTP middleware")
	if len(srv.httpMiddleware) > 0 {
		for i := len(srv.httpMiddleware) - 1; i >= 0; i-- {
			mw := srv.httpMiddleware[i]
			if mw == nil {
				continue
			}
			srv.handler = mw(srv.handler)
		}
	}

	// create the HTTP server
	srv.Server = &http.Server{
		Addr: srv.hostPort,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(
			srv.handler,
			&http2.Server{},
		),
		ReadHeaderTimeout: srv.config.ReadHeaderTimeout,
		ReadTimeout:       srv.config.ReadTimeout,
		WriteTimeout:      srv.config.WriteTimeout,
		IdleTimeout:       srv.config.IdleTimeout,
		MaxHeaderBytes:    srv.config.MaxHeaderBytes,
	}

	return srv, nil
}

// RPCServer returns the underlying Connect server, e.g. for use with an
// in-process transport in tests.
func (s *ConnectServer) RPCServer() *connect.Server {
	return s.rpcServer
}
