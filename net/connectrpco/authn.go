package connectrpco

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/authn"
	connectv1 "connectrpc.com/connect"
	"connectrpc.com/connect/v2"

	"github.com/go-toho/contrib/net/httpo"
)

// NewAuthnMiddleware creates a Connect authentication middleware.
//
// The authentication function may return either an authn.Errorf error or a
// Connect v2 *connect.Error; both are written with their code and message.
func NewAuthnMiddleware(authnFn func(context.Context, *http.Request) (any, error)) httpo.HttpMiddleware {
	middleware := authn.NewMiddleware(func(ctx context.Context, req *http.Request) (any, error) {
		info, err := authnFn(ctx, req)
		return info, toConnectV1Error(err)
	})
	return middleware.Wrap
}

// toConnectV1Error converts a Connect v2 error into its v1 equivalent. authn is
// built on Connect v1 and writes any other error type as CodeUnknown.
// TODO: remove once authn supports Connect v2.
func toConnectV1Error(err error) error {
	if connectErr, ok := errors.AsType[*connect.Error](err); ok {
		return connectv1.NewError(connectv1.Code(connectErr.Code()), errors.New(connectErr.Message()))
	}
	return err
}
