package corsofx

import (
	"go.uber.org/fx"

	"github.com/go-toho/contrib/middleware/corso"
	"github.com/go-toho/toho/pkg/fxtags"
)

var Module = fx.Module("cors",
	provideConfigPointer,
	provideConfig,
)

var (
	provideConfigPointer = fx.Provide(
		fx.Annotate(
			func(config any) *corso.Config {
				if config != nil {
					switch v := config.(type) {
					case *corso.Config:
						return v
					case corso.Config:
						return &v
					default:
						break
					}
				}
				return &corso.Config{}
			},
			fx.ParamTags(fxtags.NamedOptional(corso.NamedConfig)),
			fx.ResultTags(fxtags.Named(corso.NamedConfig)),
		),
	)

	provideConfig = fx.Provide(
		fx.Annotate(
			func(config *corso.Config) corso.Config {
				return *config
			},
			fx.ParamTags(fxtags.Named(corso.NamedConfig)),
			fx.ResultTags(fxtags.Named(corso.NamedConfig)),
		),
	)
)
