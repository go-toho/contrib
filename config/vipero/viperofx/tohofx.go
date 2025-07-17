package viperofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/toho/tohofx"
)

func init() {
	tohofx.Add("github.com/spf13/viper", func() fx.Option {
		return Module
	})
}
