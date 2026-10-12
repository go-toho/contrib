package connectrpco

import (
	"context"
	"net/http"

	"connectrpc.com/connect/v2/connecthttp"

	"github.com/go-toho/contrib/net/connectrpco/internal/authn"
	"github.com/go-toho/contrib/net/httpo"
)

// NewAuthnMiddleware creates a Connect authentication middleware. Pass the same
// Connect HTTP options as the server.
func NewAuthnMiddleware(
	authnFn func(context.Context, *http.Request) (any, error),
	opts ...connecthttp.Option,
) httpo.HttpMiddleware {
	return authn.NewMiddleware(authnFn, opts...).Wrap
}

// GetAuthnInfo returns the authentication information attached to the context.
func GetAuthnInfo(ctx context.Context) any {
	return authn.GetInfo(ctx)
}

// SetAuthnInfo attaches authentication information to the context.
func SetAuthnInfo(ctx context.Context, info any) context.Context {
	return authn.SetInfo(ctx, info)
}
