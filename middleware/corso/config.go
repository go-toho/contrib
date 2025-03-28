package corso

import "github.com/rs/cors"

type Config struct {
	// Enable CORS
	// If set to true, CORS will be enabled and preflight-requests (OPTION) will be answered.
	Enabled bool

	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string `default:"*"`
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string `default:"HEAD,GET,POST"`
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [].
	AllowedHeaders []string
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Default value is 0, which stands for no
	// Access-Control-Max-Age header to be sent back, resulting in browsers
	// using their default value (5s by spec). If you need to force a 0 max-age,
	// set `MaxAge` to a negative value (ie: -1).
	MaxAge int `default:"0"`
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// AllowPrivateNetwork indicates whether to accept cross-origin requests over a
	// private network.
	AllowPrivateNetwork bool
	// OptionsPassthrough instructs preflight to let other potential next handlers to
	// process the OPTIONS method. Turn this on if your application handles OPTIONS.
	OptionsPassthrough bool
	// Provides a status code to use for successful OPTIONS requests.
	// Default value is http.StatusNoContent (204).
	OptionsSuccessStatus int `default:"204"`
	// Debugging flag adds additional output to debug server side CORS issues
	Debug bool
}

func (c *Config) CORSOptions() (cors.Options, bool) {
	return cors.Options{
		AllowedOrigins:       c.AllowedOrigins,
		AllowedMethods:       c.AllowedMethods,
		AllowedHeaders:       c.AllowedHeaders,
		ExposedHeaders:       c.ExposedHeaders,
		MaxAge:               c.MaxAge,
		AllowCredentials:     c.AllowCredentials,
		AllowPrivateNetwork:  c.AllowPrivateNetwork,
		OptionsPassthrough:   c.OptionsPassthrough,
		OptionsSuccessStatus: c.OptionsSuccessStatus,
		Debug:                c.Debug,
	}, c.Enabled
}
