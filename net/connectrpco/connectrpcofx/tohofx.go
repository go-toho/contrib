package connectrpcofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("github.com/connectrpc", func() fx.Option {
		return Module
	})
}
