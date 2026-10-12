package connectrpcofx

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/go-toho/contrib/net/connectrpco"
	"github.com/go-toho/contrib/net/connectrpco/internal/echotest"
	"github.com/go-toho/contrib/net/httpo"
	"github.com/go-toho/toho/pkg/fxtags"
)

type testPrinter struct{ t *testing.T }

func (p testPrinter) Printf(msg string, args ...any) {
	p.t.Log(append([]any{msg}, args...)...)
}

type echoService struct{}

func registerEchoService(server *connect.Server, _ *echoService) {
	echotest.Register(server)
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

func supplyConfig(config connectrpco.ConnectConfig) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() any { return config },
			fx.ResultTags(fxtags.Named(connectrpco.NamedConnectConfig)),
		),
	)
}

func startApp(t *testing.T, config connectrpco.ConnectConfig, opts ...fx.Option) {
	t.Helper()

	app := fxtest.New(t,
		Module,
		supplyConfig(config),
		fx.Provide(func() fx.Printer { return testPrinter{t} }),
		fx.Options(opts...),
	)
	app.RequireStart()
	t.Cleanup(app.RequireStop)
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

func TestModule(t *testing.T) {
	addr := freeAddr(t)
	baseURL := "http://" + addr

	var interceptorCalls, middlewareCalls atomic.Int32

	startApp(t,
		connectrpco.ConnectConfig{Enabled: true, Addr: addr},
		fx.Supply(&echoService{}),
		ProvideServiceHandler(registerEchoService),
		WithGrpcHealthHandler,
		WithGrpcReflectHandler,
		WithOtelConnectInterceptor(),
		SupplyServerInterceptors(func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				interceptorCalls.Add(1)
				return next(ctx, spec, stream)
			}
		}),
		SupplyConnectHttpOptions(connecthttp.WithReadMaxBytes(64)),
		SupplyHttpHandlers(httpo.HttpPatternHandler{
			Pattern: "/plain",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "plain")
			}),
		}),
		SupplyHttpMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCalls.Add(1)
				next.ServeHTTP(w, r)
			})
		}),
	)

	// the server starts listening asynchronously
	require.Eventually(t, func() bool {
		res, err := http.Get(baseURL + "/plain")
		if err != nil {
			return false
		}
		_ = res.Body.Close()
		return res.StatusCode == http.StatusOK
	}, 5*time.Second, 10*time.Millisecond)

	t.Run("service handler", func(t *testing.T) {
		status, body := echo(t, baseURL, `"hello"`)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, `"hello"`, body)
	})

	t.Run("handler options", func(t *testing.T) {
		status, body := echo(t, baseURL, `"`+strings.Repeat("x", 128)+`"`)
		assert.Equal(t, http.StatusTooManyRequests, status)
		assert.Contains(t, body, "resource_exhausted")
	})

	t.Run("health handler", func(t *testing.T) {
		client := grpchealth.NewClient(connect.NewClient(connecthttp.NewTransport(http.DefaultClient, baseURL)))

		res, err := client.Check(t.Context(), &grpchealth.CheckRequest{Service: echotest.ServiceName})
		require.NoError(t, err)
		assert.Equal(t, grpchealth.StatusServing, res.Status)
	})

	assert.Positive(t, interceptorCalls.Load())
	assert.Positive(t, middlewareCalls.Load())
}

func TestModule_UtilityRegisterFuncs(t *testing.T) {
	var procedures []string

	// the server is built when the app is created, so it does not need to start
	fxtest.New(t,
		Module,
		supplyConfig(connectrpco.ConnectConfig{Enabled: true, Addr: freeAddr(t)}),
		fx.Provide(func() fx.Printer { return testPrinter{t} }),
		// supplied before the application service on purpose
		SupplyUtilityRegisterFuncs(func(server *connect.Server) {
			for spec := range server.Specs() {
				procedures = append(procedures, spec.Procedure)
			}
		}),
		SupplyRegisterFuncs(echotest.Register),
	)

	assert.Equal(t, []string{echotest.Procedure}, procedures)
}

func TestModule_Disabled(t *testing.T) {
	addr := freeAddr(t)

	startApp(t,
		connectrpco.ConnectConfig{Enabled: false, Addr: addr},
		SupplyRegisterFuncs(echotest.Register),
	)

	_, err := http.Get("http://" + addr + "/")
	assert.Error(t, err)
}
