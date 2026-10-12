package connectrpco

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"connectrpc.com/grpcreflect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-toho/contrib/net/connectrpco/internal/echotest"
)

func newTestServer(t *testing.T, opts ...ConnectServerOption) *httptest.Server {
	t.Helper()

	srv, err := NewConnectServer(append([]ConnectServerOption{Address("127.0.0.1:0")}, opts...)...)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}

// newH2CClient returns an HTTP client that speaks HTTP/2 without TLS, which
// streaming RPCs such as reflection require.
func newH2CClient() *http.Client {
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: &http.Transport{Protocols: protocols}}
}

func echo(t *testing.T, baseURL, body string) (int, string) {
	t.Helper()

	res, err := http.Post(baseURL+echotest.Procedure, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, string(data)
}

func TestServiceNames(t *testing.T) {
	server := connect.NewServer()
	echotest.Register(server)
	grpchealth.Register(server, grpchealth.NewStaticChecker())

	assert.Equal(t, []string{echotest.ServiceName, grpchealth.HealthV1ServiceName}, serviceNames(server))
}

func TestConnectServer_ServiceRegistrars(t *testing.T) {
	ts := newTestServer(t, ServiceRegistrars(echotest.Register, nil))

	status, body := echo(t, ts.URL, `"hello"`)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, `"hello"`, body)
}

func TestConnectServer_ServerInterceptors(t *testing.T) {
	var calls []string
	interceptor := func(name string) connect.ServerInterceptor {
		return func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				calls = append(calls, name+":"+spec.Procedure)
				return next(ctx, spec, stream)
			}
		}
	}

	ts := newTestServer(t,
		ServiceRegistrars(echotest.Register),
		ServerInterceptors(interceptor("first"), interceptor("second")),
	)

	status, _ := echo(t, ts.URL, `"hello"`)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, []string{
		"first:/" + echotest.ServiceName + "/Echo",
		"second:/" + echotest.ServiceName + "/Echo",
	}, calls)
}

func TestConnectServer_HandlerOptions(t *testing.T) {
	ts := newTestServer(t,
		ServiceRegistrars(echotest.Register),
		HandlerOptions(connecthttp.WithReadMaxBytes(8)),
	)

	status, body := echo(t, ts.URL, `"this message is too long"`)
	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.Contains(t, body, "resource_exhausted")
}

func TestConnectServer_HttpHandlersAndMiddleware(t *testing.T) {
	var middlewareCalls atomic.Int32

	ts := newTestServer(t,
		ServiceRegistrars(echotest.Register),
		Handle("/plain", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "plain")
		})),
		HttpMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCalls.Add(1)
				next.ServeHTTP(w, r)
			})
		}),
	)

	res, err := http.Get(ts.URL + "/plain")
	require.NoError(t, err)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, "plain", string(data))

	status, _ := echo(t, ts.URL, `"hello"`)
	assert.Equal(t, http.StatusOK, status)

	assert.Equal(t, int32(2), middlewareCalls.Load())
}

func TestGrpcHealthRegistrar(t *testing.T) {
	ts := newTestServer(t, ServiceRegistrars(
		echotest.Register,
		NewGrpcReflectRegistrar(),
		NewGrpcHealthRegistrar(),
	))

	client := grpchealth.NewClient(connect.NewClient(connecthttp.NewTransport(http.DefaultClient, ts.URL)))

	tests := []struct {
		service string
		status  grpchealth.Status
		code    connect.Code
	}{
		{service: "", status: grpchealth.StatusServing},
		{service: echotest.ServiceName, status: grpchealth.StatusServing},
		{service: grpcreflect.ReflectV1ServiceName, code: connect.CodeNotFound},
		{service: "unknown.v1.Service", code: connect.CodeNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			res, err := client.Check(t.Context(), &grpchealth.CheckRequest{Service: tt.service})
			if tt.code != 0 {
				assert.Equal(t, tt.code, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.status, res.Status)
		})
	}
}

func TestGrpcHealthRegistrar_NoServices(t *testing.T) {
	server := connect.NewServer()
	NewGrpcHealthRegistrar()(server)

	assert.Empty(t, serviceNames(server))
}

func TestGrpcReflectRegistrar(t *testing.T) {
	ts := newTestServer(t, ServiceRegistrars(
		NewGrpcReflectRegistrar(),
		echotest.Register,
	))

	client := grpcreflect.NewClient(connect.NewClient(connecthttp.NewTransport(newH2CClient(), ts.URL, connecthttp.WithGRPC())))
	stream := client.NewStream(t.Context())
	t.Cleanup(func() { _ = stream.Close() })

	services, err := stream.ListServices()
	require.NoError(t, err)

	var names []string
	for _, service := range services {
		names = append(names, string(service))
	}
	assert.ElementsMatch(t, []string{
		echotest.ServiceName,
		grpcreflect.ReflectV1ServiceName,
		grpcreflect.ReflectV1AlphaServiceName,
	}, names)
}
