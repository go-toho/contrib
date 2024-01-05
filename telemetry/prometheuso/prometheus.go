package prometheuso

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func DefaultRegistry() (prometheus.Registerer, prometheus.Gatherer) {
	return prometheus.DefaultRegisterer, prometheus.DefaultGatherer
}

func HttpHandler(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}
