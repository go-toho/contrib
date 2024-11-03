package httpo

import "net/http"

type HttpPatternHandler struct {
	Pattern string
	Handler http.Handler
}

type HttpMiddleware func(http.Handler) http.Handler
