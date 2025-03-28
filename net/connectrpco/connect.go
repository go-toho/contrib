package connectrpco

import (
	"fmt"
	"net"
	"net/http"
	"time"

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

// Handle registers a new handler for a given pattern.
func Handle(pattern string, handler http.Handler) ConnectServerOption {
	return func(s *ConnectServer) {
		s.connectHandlers = append(s.connectHandlers, httpo.HttpPatternHandler{
			Pattern: pattern,
			Handler: handler,
		})
	}
}

// HttpPatternHandlers registers handlers.
func HttpPatternHandlers(patternHandlers ...httpo.HttpPatternHandler) ConnectServerOption {
	return func(s *ConnectServer) {
		s.connectHandlers = append(s.connectHandlers, patternHandlers...)
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

	config          *ConnectConfig
	connectHandlers []httpo.HttpPatternHandler
	httpMiddleware  []httpo.HttpMiddleware

	hostPort string
	mux      *http.ServeMux
	handler  http.Handler
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

	// create a new mux
	srv.mux = http.NewServeMux()

	// add connect handlers
	if len(srv.connectHandlers) > 0 {
		for _, handler := range srv.connectHandlers {
			srv.mux.Handle(handler.Pattern, handler.Handler)
		}
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
