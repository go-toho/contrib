package connectrpcofx

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"go.uber.org/fx"

	"github.com/go-toho/contrib/middleware/corso"
	"github.com/go-toho/contrib/net/connectrpco"
	"github.com/go-toho/contrib/net/httpo"
	"github.com/go-toho/toho/pkg/fxtags"
)

const (
	groupAllConnectHandlers = "connectrpcofx.AllConnectHandlers"
)

var Module = fx.Module("connectrpc",
	provideConnectConfigPointer,
	provideConnectConfig,
	provideConnectServiceNames,
	provideAllConnectHandlers,
	invokeServer,
)

var (
	provideConnectConfigPointer = fx.Provide(
		fx.Annotate(
			func(config any) *connectrpco.ConnectConfig {
				if config != nil {
					switch v := config.(type) {
					case *connectrpco.ConnectConfig:
						return v
					case connectrpco.ConnectConfig:
						return &v
					default:
						break
					}
				}
				return &connectrpco.ConnectConfig{}
			},
			fx.ParamTags(fxtags.NamedOptional(connectrpco.NamedConnectConfig)),
			fx.ResultTags(fxtags.Named(connectrpco.NamedConnectConfig)),
		),
	)

	provideConnectConfig = fx.Provide(
		fx.Annotate(
			func(config *connectrpco.ConnectConfig) connectrpco.ConnectConfig {
				return *config
			},
			fx.ParamTags(fxtags.Named(connectrpco.NamedConnectConfig)),
			fx.ResultTags(fxtags.Named(connectrpco.NamedConnectConfig)),
		),
	)

	provideConnectServiceNames = fx.Provide(
		fx.Annotate(
			func(connectHandlers []httpo.HttpPatternHandler) []string {
				var services []string
				// extract service names
				for _, handlers := range connectHandlers {
					services = append(services, strings.Trim(handlers.Pattern, "/"))
				}
				return services
			},
			fx.ParamTags(fxtags.Group(connectrpco.GroupConnectHandler)),
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupConnectServiceNames)),
		),
	)

	provideAllConnectHandlers = fx.Provide(
		fx.Annotate(
			func(connectHandlers, utilityConnectHandlers []httpo.HttpPatternHandler) []httpo.HttpPatternHandler {
				var handlers []httpo.HttpPatternHandler
				handlers = append(handlers, connectHandlers...)
				handlers = append(handlers, utilityConnectHandlers...)
				return handlers
			},
			fx.ParamTags(
				fxtags.Group(connectrpco.GroupConnectHandler),
				fxtags.Group(connectrpco.GroupUtilityConnectHandler),
			),
			fx.ResultTags(fxtags.GroupFlatten(groupAllConnectHandlers)),
		),
	)

	invokeServer = fx.Invoke(
		fx.Annotate(
			NewConnectServer,
			fx.ParamTags(
				fxtags.Named(connectrpco.NamedConnectConfig),
				fxtags.Group(groupAllConnectHandlers),
				fxtags.Group(connectrpco.GroupHttpMiddleware),
			),
		),
	)
)

func NewConnectServer(
	config connectrpco.ConnectConfig,
	connectHandlers []httpo.HttpPatternHandler,
	httpMiddleware []httpo.HttpMiddleware,
	log fx.Printer,
	lifecycle fx.Lifecycle,
) error {
	if !config.Enabled {
		log.Printf("Connect server not enabled")
		return nil
	}

	server, err := connectrpco.NewConnectServer(
		connectrpco.WithConnectConfig(config),
		connectrpco.HttpPatternHandlers(connectHandlers...),
		connectrpco.HttpMiddleware(httpMiddleware...),
	)
	if err != nil {
		return err
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				log.Printf("starting Connect server",
					"address", fmt.Sprintf("http://%s/", server.Addr),
				)

				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("unable to start Connect server", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Printf("stopping Connect server")
			return server.Shutdown(ctx)
		},
	})

	return nil
}

var WithGrpcHealthHandler = fx.Provide(
	fx.Annotate(
		connectrpco.NewGrpcHealthHandler,
		fx.ParamTags(
			fxtags.Group(connectrpco.GroupConnectServiceNames),
			fxtags.Group(connectrpco.GroupConnectHandlerOptions),
		),
		fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupUtilityConnectHandler)),
	),
)

var WithGrpcReflectHandler = fx.Provide(
	fx.Annotate(
		connectrpco.NewGrpcReflectHandler,
		fx.ParamTags(
			fxtags.Group(connectrpco.GroupConnectServiceNames),
			fxtags.Group(connectrpco.GroupConnectHandlerOptions),
		),
		fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupUtilityConnectHandler)),
	),
)

var WithCorsHttpMiddleware = fx.Provide(
	fx.Annotate(
		connectrpco.NewCorsHandler,
		fx.ParamTags(fxtags.Named(corso.NamedConfig)),
		fx.ResultTags(fxtags.Group(connectrpco.GroupHttpMiddleware)),
	),
)

var WithOtelConnectInterceptor = func(trustRemote bool, omitTraceEvents bool) fx.Option {
	var opts []otelconnect.Option

	if trustRemote {
		opts = append(opts, otelconnect.WithTrustRemote())
	}

	if omitTraceEvents {
		opts = append(opts, otelconnect.WithoutTraceEvents())
	}

	return fx.Provide(
		fx.Annotate(
			func() (connect.HandlerOption, error) {
				otelInterceptor, err := otelconnect.NewInterceptor(opts...)
				if err != nil {
					return nil, err
				}
				return connect.WithInterceptors(otelInterceptor), nil
			},
			fx.ResultTags(fxtags.Group(connectrpco.GroupConnectHandlerOptions)),
		),
	)
}

func SupplyConnectHandlerOptions(opts ...connect.HandlerOption) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connect.HandlerOption {
				return opts
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupConnectHandlerOptions)),
		),
	)
}

func SupplyConnectHandler(fn func(opts ...connect.HandlerOption) (string, http.Handler)) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(opts ...connect.HandlerOption) httpo.HttpPatternHandler {
				path, handler := fn(opts...)
				return httpo.HttpPatternHandler{
					Pattern: path,
					Handler: handler,
				}
			},
			fx.ParamTags(fxtags.Group(connectrpco.GroupConnectHandlerOptions)),
			fx.ResultTags(fxtags.Group(connectrpco.GroupConnectHandler)),
		),
	)
}

func SupplyHttpMiddleware(middleware ...httpo.HttpMiddleware) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []httpo.HttpMiddleware {
				return middleware
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupHttpMiddleware)),
		),
	)
}
