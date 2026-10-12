package connectrpcofx

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/otelconnect"
	"go.uber.org/fx"

	"github.com/go-toho/contrib/middleware/corso"
	"github.com/go-toho/contrib/net/connectrpco"
	"github.com/go-toho/contrib/net/httpo"
	"github.com/go-toho/toho/pkg/fxtags"
)

var Module = fx.Module("connectrpc",
	provideConnectConfigPointer,
	provideConnectConfig,
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

	invokeServer = fx.Invoke(
		fx.Annotate(
			NewConnectServer,
			fx.ParamTags(
				fxtags.Named(connectrpco.NamedConnectConfig),
				fxtags.Group(connectrpco.GroupServiceRegistrar),
				fxtags.Group(connectrpco.GroupUtilityServiceRegistrar),
				fxtags.Group(connectrpco.GroupServerInterceptor),
				fxtags.Group(connectrpco.GroupHandlerOption),
				fxtags.Group(connectrpco.GroupHttpHandler),
				fxtags.Group(connectrpco.GroupHttpMiddleware),
			),
		),
	)
)

func NewConnectServer(
	config connectrpco.ConnectConfig,
	serviceRegistrars []connectrpco.ServiceRegistrar,
	utilityServiceRegistrars []connectrpco.ServiceRegistrar,
	serverInterceptors []connect.ServerInterceptor,
	handlerOptions []connecthttp.Option,
	httpHandlers []httpo.HttpPatternHandler,
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
		connectrpco.ServerInterceptors(serverInterceptors...),
		// utility services (health, reflection) inspect the application
		// services, so they must be registered after them
		connectrpco.ServiceRegistrars(serviceRegistrars...),
		connectrpco.ServiceRegistrars(utilityServiceRegistrars...),
		connectrpco.HandlerOptions(handlerOptions...),
		connectrpco.HttpPatternHandlers(httpHandlers...),
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
		connectrpco.NewGrpcHealthRegistrar,
		fx.ResultTags(fxtags.Group(connectrpco.GroupUtilityServiceRegistrar)),
	),
)

var WithGrpcReflectHandler = fx.Provide(
	fx.Annotate(
		func() connectrpco.ServiceRegistrar {
			return connectrpco.NewGrpcReflectRegistrar()
		},
		fx.ResultTags(fxtags.Group(connectrpco.GroupUtilityServiceRegistrar)),
	),
)

var WithCorsHttpMiddleware = fx.Provide(
	fx.Annotate(
		connectrpco.NewCorsHandler,
		fx.ParamTags(fxtags.Named(corso.NamedConfig)),
		fx.ResultTags(fxtags.Group(connectrpco.GroupHttpMiddleware)),
	),
)

// WithOtelConnectInterceptor provides a server interceptor configured with otelconnect options.
var WithOtelConnectInterceptor = func(opts ...otelconnect.Option) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() (connect.ServerInterceptor, error) {
				return otelconnect.NewServerInterceptor(opts...)
			},
			fx.ResultTags(fxtags.Group(connectrpco.GroupServerInterceptor)),
		),
	)
}

// ProvideServiceHandler registers a Connect service whose implementation is
// resolved from the container, e.g.
//
//	ProvideServiceHandler(pingv1connect.RegisterPingServiceHandler)
//
// requires a pingv1connect.PingServiceHandler to be provided.
func ProvideServiceHandler[T any](register func(*connect.Server, T)) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(svc T) connectrpco.ServiceRegistrar {
				return func(server *connect.Server) {
					register(server, svc)
				}
			},
			fx.ResultTags(fxtags.Group(connectrpco.GroupServiceRegistrar)),
		),
	)
}

func SupplyServiceRegistrars(registrars ...connectrpco.ServiceRegistrar) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connectrpco.ServiceRegistrar {
				return registrars
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupServiceRegistrar)),
		),
	)
}

func SupplyServerInterceptors(interceptors ...connect.ServerInterceptor) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connect.ServerInterceptor {
				return interceptors
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupServerInterceptor)),
		),
	)
}

func SupplyHandlerOptions(opts ...connecthttp.Option) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connecthttp.Option {
				return opts
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupHandlerOption)),
		),
	)
}

func SupplyHttpHandlers(handlers ...httpo.HttpPatternHandler) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []httpo.HttpPatternHandler {
				return handlers
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupHttpHandler)),
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
