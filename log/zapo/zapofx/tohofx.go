package zapofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("go.uber.org/zap", func() fx.Option {
		return Module
	})
}
