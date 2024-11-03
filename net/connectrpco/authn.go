package connectrpco

import (
	"context"
	"net/http"

	"connectrpc.com/authn"

	"github.com/go-toho/contrib/net/httpo"
)

// NewAuthnMiddleware creates a Connect authentication middleware.
func NewAuthnMiddleware(authnFn func(context.Context, *http.Request) (any, error)) httpo.HttpMiddleware {
	middleware := authn.NewMiddleware(authnFn)
	return middleware.Wrap
}
