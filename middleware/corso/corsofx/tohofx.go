package corsofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("github.com/rs/cors", func() fx.Option {
		return Module
	})
}
