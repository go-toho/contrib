package validatorofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/contrib/validation/validatoro"
)

var Module = fx.Module("validator",
	fx.Provide(validatoro.New),
)
