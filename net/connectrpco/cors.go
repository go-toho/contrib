package connectrpco

import (
	"slices"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"

	"github.com/go-toho/contrib/middleware/corso"
	"github.com/go-toho/contrib/net/httpo"
)

// NewConnectCorsOptions creates a Connect CORS options.
// It merges the provided options with the defaults required by Connect.
func NewConnectCorsOptions(options cors.Options) cors.Options {
	for _, method := range connectcors.AllowedMethods() {
		if !slices.Contains(options.AllowedMethods, method) {
			options.AllowedMethods = append(options.AllowedMethods, method)
		}
	}

	for _, header := range connectcors.AllowedHeaders() {
		if !slices.Contains(options.AllowedHeaders, header) {
			options.AllowedHeaders = append(options.AllowedHeaders, header)
		}
	}

	for _, header := range connectcors.ExposedHeaders() {
		if !slices.Contains(options.ExposedHeaders, header) {
			options.ExposedHeaders = append(options.ExposedHeaders, header)
		}
	}

	if options.MaxAge == 0 {
		options.MaxAge = 7200 // 2 hours in seconds
	}

	return options
}

// NewCorsHandler creates a Connect CORS middleware for Connect HTTP handler.
func NewCorsHandler(config corso.Config) httpo.HttpMiddleware {
	options, enabled := config.CORSOptions()
	if !enabled {
		return nil
	}

	options = NewConnectCorsOptions(options)

	c := cors.New(options)
	return c.Handler
}
