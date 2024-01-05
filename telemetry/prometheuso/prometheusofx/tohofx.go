package prometheusofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("github.com/prometheus/client_golang", func() fx.Option {
		return Module
	})
}
