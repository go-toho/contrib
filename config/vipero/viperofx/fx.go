package viperofx

import (
	"strings"

	"github.com/go-toho/toho/app"
	"github.com/go-toho/toho/config"
	"github.com/go-toho/toho/pkg/fxtags"
	"github.com/spf13/viper"
	"go.uber.org/fx"

	"github.com/go-toho/contrib/config/vipero"
)

var Module = fx.Module("viper",
	provideViper,
	provideConfig,
)

var (
	provideViper = fx.Provide(
		fx.Annotate(
			func(
				appName string,
				v *viper.Viper,
			) (*viper.Viper, error) {
				if v == nil {
					v = vipero.New(appName)
				} else {
					if appName != "" {
						v.SetDefault(app.NamedAppName, appName)
						v.SetEnvPrefix(strings.ToUpper(appName))
					}
				}
				return v, nil
			},
			fx.ParamTags(
				fxtags.NamedOptional(app.NamedAppName),
				fxtags.NamedOptional(vipero.NamedViper),
			),
		),
	)

	provideConfig = fx.Provide(
		fx.Annotate(
			func(
				v *viper.Viper,
				cfg any,
				files []string,
				opts []viper.DecoderConfigOption,
			) (any, error) {
				if err := vipero.LoadConfig(v, cfg, files, opts...); err != nil {
					return nil, err
				}
				return cfg, nil
			},
			fx.ParamTags(
				fxtags.Empty,
				fxtags.Named(config.NamedConfigPointerIn),
				fxtags.Group(config.GroupConfigFiles),
				fxtags.Group(vipero.GroupDecoderConfigOptions),
			),
			fx.ResultTags(fxtags.Named(config.NamedConfigPointerOut)),
		),
	)
)

func SupplyViper(v *viper.Viper) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() *viper.Viper {
				return v
			},
			fx.ResultTags(fxtags.Named(vipero.NamedViper)),
		),
	)
}

func SupplyDecoderConfigOption(option viper.DecoderConfigOption) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() viper.DecoderConfigOption {
				return option
			},
			fx.ResultTags(fxtags.Group(vipero.GroupDecoderConfigOptions)),
		),
	)
}
