package connectrpcotest_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"connectrpc.com/grpcreflect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-toho/contrib/net/connectrpco"
	"github.com/go-toho/contrib/net/connectrpco/connectrpcotest"
	"github.com/go-toho/contrib/net/connectrpco/internal/echotest"
)

func TestServer_Client(t *testing.T) {
	var serverCalls, clientCalls int

	srv := connectrpcotest.NewServer(t,
		connectrpco.RegisterServices(echotest.Register, connectrpco.RegisterGrpcHealth),
		connectrpco.ServerInterceptors(func(next connect.ServerFunc) connect.ServerFunc {
			return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
				serverCalls++
				return next(ctx, spec, stream)
			}
		}),
	)

	client := grpchealth.NewClient(srv.Client(func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			clientCalls++
			return next(ctx, spec)
		}
	}))

	res, err := client.Check(t.Context(), &grpchealth.CheckRequest{Service: echotest.ServiceName})
	require.NoError(t, err)
	assert.Equal(t, grpchealth.StatusServing, res.Status)
	assert.Equal(t, 1, serverCalls)
	assert.Equal(t, 1, clientCalls)
}

func TestServer_Streaming(t *testing.T) {
	srv := connectrpcotest.NewServer(t,
		connectrpco.RegisterServices(echotest.Register, connectrpco.RegisterGrpcReflect),
	)

	// reflection is a bidirectional stream, which requires HTTP/2
	client := grpcreflect.NewClient(connect.NewClient(srv.Transport(connecthttp.WithGRPC())))
	stream := client.NewStream(t.Context())
	t.Cleanup(func() { _ = stream.Close() })

	services, err := stream.ListServices()
	require.NoError(t, err)
	assert.Contains(t, services, protoreflect.FullName(echotest.ServiceName))
}

func TestServer_HttpClient(t *testing.T) {
	srv := connectrpcotest.NewServer(t,
		connectrpco.RegisterServices(echotest.Register),
	)

	res, err := srv.HttpClient().Post(srv.URL+echotest.Procedure, "application/json", strings.NewReader(`"hello"`))
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "HTTP/2.0", res.Proto)
	assert.Equal(t, `"hello"`, string(body))
}
