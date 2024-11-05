package validatorofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("github.com/go-playground/validator", func() fx.Option {
		return Module
	})
}
