package prometheusofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/contrib/telemetry/prometheuso"
)

var Module = fx.Module("prometheus",
	providePrometheusRegistry,
)

var (
	providePrometheusRegistry = fx.Provide(prometheuso.DefaultRegistry)
)
