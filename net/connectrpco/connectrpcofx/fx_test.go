package connectrpcofx_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/authn"
	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"connectrpc.com/grpcreflect/v2"
	"connectrpc.com/otelconnect"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/go-toho/contrib/middleware/corso"
	"github.com/go-toho/contrib/net/connectrpco"
	"github.com/go-toho/contrib/net/connectrpco/connectrpcofx"
	"github.com/go-toho/contrib/net/httpo"
	"github.com/go-toho/toho/pkg/fxtags"
)

const echoProcedure = "/test.Echo/Echo"

func registerEcho(server *connect.Server) {
	for _, procedure := range []string{echoProcedure, "/test.Echo/Again"} {
		server.Register(connect.Method{
			Spec: connect.Spec{Procedure: procedure, StreamType: connect.StreamTypeUnary},
			Handler: func(ctx context.Context, _ connect.Spec, stream connect.ServerStream) error {
				var message wrapperspb.StringValue
				if err := stream.Receive(&message); err != nil {
					return err
				}
				if identity, ok := authn.GetInfo(ctx).(string); ok {
					info, _ := connect.CallInfoForServerContext(ctx)
					info.ResponseHeader().Set("X-Identity", identity)
				}
				return stream.Send(&message)
			},
		})
	}
}

func newApp(t *testing.T, config connectrpco.ConnectConfig, options ...fx.Option) (*fx.App, *connectrpco.ConnectServer) {
	t.Helper()
	var server *connectrpco.ConnectServer
	app := fx.New(
		fx.NopLogger,
		connectrpcofx.Module,
		fx.Supply(fx.Annotate(log.New(io.Discard, "", 0), fx.As(new(fx.Printer)))),
		fx.Supply(fx.Annotate(config, fx.As(new(any)), fx.ResultTags(fxtags.Named(connectrpco.NamedConnectConfig)))),
		fx.Populate(&server),
		fx.Options(options...),
	)
	return app, server
}

func newHTTPServer(t *testing.T, options ...fx.Option) *httptest.Server {
	t.Helper()
	app, server := newApp(t, connectrpco.ConnectConfig{Enabled: true, Addr: "127.0.0.1:0"}, options...)
	require.NoError(t, app.Err())
	require.NotNil(t, server)
	// Exercise the assembled HTTP handler without starting Fx's separate listener.
	httpServer := httptest.NewUnstartedServer(server.Handler)
	httpServer.EnableHTTP2 = true
	httpServer.StartTLS()
	t.Cleanup(httpServer.Close)
	return httpServer
}

func TestModuleRPCAndMiddleware(t *testing.T) {
	var intercepted, observed atomic.Int32
	server := newHTTPServer(t,
		connectrpcofx.SupplyConnectHandler(registerEcho),
		connectrpcofx.SupplyConnectInterceptors(func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				intercepted.Add(1)
				info, _ := connect.CallInfoForServerContext(ctx)
				info.ResponseHeader().Add("X-Interceptor", "first")
				return next(ctx, spec, stream)
			}
		}, func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				info, _ := connect.CallInfoForServerContext(ctx)
				info.ResponseHeader().Add("X-Interceptor", "second")
				return next(ctx, spec, stream)
			}
		}),
		connectrpcofx.WithOtelConnectInterceptor(otelconnect.WithFilter(func(context.Context, connect.Spec) bool {
			observed.Add(1)
			return false
		})),
		connectrpcofx.SupplyHttpMiddleware(connectrpco.NewAuthnMiddleware(func(_ context.Context, request *http.Request) (any, error) {
			if token, ok := authn.BearerToken(request); !ok || token != "valid" {
				return nil, authn.Errorf("missing credentials")
			}
			return "test-user", nil
		})),
		connectrpcofx.WithCorsHttpMiddleware,
		fx.Supply(fx.Annotate(corso.Config{
			Enabled: true, AllowedOrigins: []string{"https://example.com"},
		}, fx.ResultTags(fxtags.Named(corso.NamedConfig)))),
	)

	for _, protocol := range []struct {
		name    string
		options []connecthttp.Option
	}{
		{name: "connect"},
		{name: "grpc", options: []connecthttp.Option{connecthttp.WithGRPC()}},
		{name: "grpc-web", options: []connecthttp.Option{connecthttp.WithGRPCWeb()}},
	} {
		t.Run(protocol.name, func(t *testing.T) {
			client := connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL, protocol.options...))
			spec := connect.Spec{Procedure: echoProcedure, StreamType: connect.StreamTypeUnary}
			var response wrapperspb.StringValue
			err := client.CallUnary(t.Context(), spec, wrapperspb.String("hello"), &response)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

			ctx, info := connect.NewClientContext(t.Context())
			info.RequestHeader().Set("Authorization", "Bearer valid")
			info.RequestHeader().Set("Origin", "https://example.com")
			require.NoError(t, client.CallUnary(ctx, spec, wrapperspb.String("hello"), &response))
			require.Equal(t, "hello", response.Value)
			require.Equal(t, "test-user", info.ResponseHeader().Get("X-Identity"))
			require.Equal(t, []string{"first", "second"}, info.ResponseHeader().Values("X-Interceptor"))
			require.Equal(t, "https://example.com", info.ResponseHeader().Get("Access-Control-Allow-Origin"))
		})
	}
	require.EqualValues(t, 3, intercepted.Load())
	require.EqualValues(t, 3, observed.Load())
}

func TestModuleHealthAndReflection(t *testing.T) {
	var intercepted atomic.Int32
	plainHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := newHTTPServer(t,
		fx.Provide(fx.Annotate(func() httpo.HttpPatternHandler {
			return httpo.HttpPatternHandler{Pattern: "/metrics", Handler: plainHandler}
		}, fx.ResultTags(fxtags.Group(connectrpco.GroupConnectHandler)))),
		fx.Provide(fx.Annotate(func() []httpo.HttpPatternHandler {
			return []httpo.HttpPatternHandler{{Pattern: "/status", Handler: plainHandler}}
		}, fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupUtilityConnectHandler)))),
		connectrpcofx.SupplyHttpMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		}),
		connectrpcofx.SupplyConnectHandler(registerEcho),
		connectrpcofx.SupplyConnectHandler(func(server *connect.Server) {
			server.Register(connect.Method{Spec: connect.Spec{Procedure: "/test.Other/Echo"}, Handler: func(context.Context, connect.Spec, connect.ServerStream) error { return nil }})
		}),
		connectrpcofx.WithGrpcHealthHandler,
		connectrpcofx.WithGrpcReflectHandler,
		connectrpcofx.SupplyConnectInterceptors(func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				intercepted.Add(1)
				return next(ctx, spec, stream)
			}
		}),
	)
	for _, path := range []string{"/metrics", "/status"} {
		response, err := server.Client().Get(server.URL + path)
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusNoContent, response.StatusCode, path)
		require.Equal(t, "applied", response.Header.Get("X-Middleware"))
	}
	client := connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL, connecthttp.WithGRPC()))
	health := grpchealth.NewClient(client)
	for _, service := range []string{"", "test.Echo", "test.Other"} {
		response, err := health.Check(t.Context(), &grpchealth.CheckRequest{Service: service})
		require.NoError(t, err)
		require.Equal(t, grpchealth.StatusServing, response.Status)
	}
	_, err := health.Check(t.Context(), &grpchealth.CheckRequest{Service: "unknown"})
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	for _, service := range []string{grpcreflect.ReflectV1ServiceName, grpcreflect.ReflectV1AlphaServiceName} {
		t.Run(service, func(t *testing.T) {
			transport := connecthttp.NewTransport(reflectionHTTPClient{server.Client(), "/" + service + "/ServerReflectionInfo"}, server.URL, connecthttp.WithGRPC())
			stream := grpcreflect.NewClient(connect.NewClient(transport)).NewStream(t.Context())
			defer stream.Close()
			services, err := stream.ListServices()
			require.NoError(t, err)
			require.ElementsMatch(t, []protoreflect.FullName{"test.Echo", "test.Other"}, services)
		})
	}
	require.EqualValues(t, 6, intercepted.Load())
}

type reflectionHTTPClient struct {
	*http.Client
	path string
}

func (c reflectionHTTPClient) Do(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.URL.Path = c.path
	return c.Client.Do(request)
}

func TestModuleReadLimits(t *testing.T) {
	for _, test := range []struct {
		name    string
		limit   []connecthttp.Option
		size    int
		blocked bool
	}{
		{name: "v2 default limit", size: 4*1024*1024 + 1, blocked: true},
		{name: "explicit limit", limit: []connecthttp.Option{connecthttp.WithReadMaxBytes(1024)}, size: 2048, blocked: true},
		{name: "explicit unlimited", limit: []connecthttp.Option{connecthttp.WithReadMaxBytes(0)}, size: 4*1024*1024 + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := newHTTPServer(t, connectrpcofx.SupplyConnectHandler(registerEcho), connectrpcofx.SupplyConnectHttpHandlerOptions(test.limit...))
			client := connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL, connecthttp.WithReadMaxBytes(0)))
			request := wrapperspb.String(strings.Repeat("x", test.size))
			var response wrapperspb.StringValue
			err := client.CallUnary(t.Context(), connect.Spec{Procedure: echoProcedure}, request, &response)
			if test.blocked {
				require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, request.Value, response.Value)
		})
	}
}

func TestModuleWithoutServices(t *testing.T) {
	server := newHTTPServer(t, connectrpcofx.WithGrpcHealthHandler, connectrpcofx.WithGrpcReflectHandler)
	for _, path := range []string{"/grpc.health.v1.Health/Check", "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"} {
		response, err := server.Client().Post(server.URL+path, "application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	}
}

func TestModuleDisabledAndInvalidAddress(t *testing.T) {
	app, server := newApp(t, connectrpco.ConnectConfig{})
	require.NoError(t, app.Err())
	require.Nil(t, server)
	require.NoError(t, app.Start(t.Context()))
	require.NoError(t, app.Stop(t.Context()))

	app, _ = newApp(t, connectrpco.ConnectConfig{Enabled: true, Addr: "[invalid"})
	require.ErrorContains(t, app.Err(), "could not resolve TCP address")
}
