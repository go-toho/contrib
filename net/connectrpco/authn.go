package connectrpco

import (
	"context"
	"net/http"

	"connectrpc.com/connect/v2/connecthttp"

	"github.com/go-toho/contrib/net/connectrpco/internal/authn"
	"github.com/go-toho/contrib/net/httpo"
)

// NewAuthnMiddleware creates HTTP middleware that authenticates requests before
// they reach the Connect handlers, so unauthenticated requests are rejected
// before the body is read.
//
// If authnFn returns an error, it is written in the format the RPC client
// expects; return a *connect.Error, e.g. with connect.CodeUnauthenticated, to
// set the code. Otherwise, any non-nil information it returns is available to
// later middleware, interceptors and handlers through GetAuthnInfo.
//
// Pass the same Connect HTTP options used for the server, so errors are
// written consistently with the handlers.
func NewAuthnMiddleware(
	authnFn func(context.Context, *http.Request) (any, error),
	opts ...connecthttp.Option,
) httpo.HttpMiddleware {
	return authn.NewMiddleware(authnFn, opts...).Wrap
}

// GetAuthnInfo returns the authentication information attached to the context
// by NewAuthnMiddleware, or nil if there is none.
func GetAuthnInfo(ctx context.Context) any {
	return authn.GetInfo(ctx)
}

// SetAuthnInfo attaches authentication information to the context, e.g. in
// tests.
func SetAuthnInfo(ctx context.Context, info any) context.Context {
	return authn.SetInfo(ctx, info)
}
