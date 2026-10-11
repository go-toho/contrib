package connectrpcofx

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

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
	provideConnectServer,
	provideHttpServer,
	fx.Invoke(func(*connectrpco.ConnectServer) {}),
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

	provideConnectServer = fx.Provide(
		fx.Annotate(
			func(registrations []func(*connect.Server), interceptorGroups [][]connect.ServerInterceptor, utilities []func(*connect.Server, []string)) *connect.Server {
				// Fx groups are unordered; preserve the order within each supplied chain.
				interceptors := slices.Concat(interceptorGroups...)
				server := connect.NewServer(interceptors...)
				for _, register := range registrations {
					register(server)
				}
				var services []string
				for spec := range server.Specs() {
					service, _, _ := strings.Cut(strings.TrimPrefix(spec.Procedure, "/"), "/")
					if !slices.Contains(services, service) {
						services = append(services, service)
					}
				}
				for _, register := range utilities {
					register(server, services)
				}
				return server
			},
			fx.ParamTags(
				fxtags.Group(connectrpco.GroupConnectHandler),
				fxtags.Group(connectrpco.GroupConnectInterceptor),
				fxtags.Group(connectrpco.GroupUtilityConnectHandler),
			),
		),
	)

	provideHttpServer = fx.Provide(
		fx.Annotate(
			NewConnectServer,
			fx.ParamTags(
				fxtags.Named(connectrpco.NamedConnectConfig),
				"",
				fxtags.Group(connectrpco.GroupConnectHandlerOptions),
				fxtags.Group(connectrpco.GroupHttpMiddleware),
			),
		),
	)
)

func NewConnectServer(
	config connectrpco.ConnectConfig,
	rpcServer *connect.Server,
	handlerOptions []connecthttp.Option,
	httpMiddleware []httpo.HttpMiddleware,
	log fx.Printer,
	lifecycle fx.Lifecycle,
) (*connectrpco.ConnectServer, error) {
	if !config.Enabled {
		log.Printf("Connect server not enabled")
		return nil, nil
	}

	server, err := connectrpco.NewConnectServer(
		connectrpco.WithConnectConfig(config),
		connectrpco.WithConnectServer(rpcServer, handlerOptions...),
		connectrpco.HttpMiddleware(httpMiddleware...),
	)
	if err != nil {
		return nil, err
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

	return server, nil
}

var WithGrpcHealthHandler = fx.Supply(
	fx.Annotate(
		connectrpco.RegisterGrpcHealthHandler,
		fx.ResultTags(fxtags.Group(connectrpco.GroupUtilityConnectHandler)),
	),
)

var WithGrpcReflectHandler = fx.Supply(
	fx.Annotate(
		connectrpco.RegisterGrpcReflectHandler,
		fx.ResultTags(fxtags.Group(connectrpco.GroupUtilityConnectHandler)),
	),
)

var WithCorsHttpMiddleware = fx.Provide(
	fx.Annotate(
		connectrpco.NewCorsHandler,
		fx.ParamTags(fxtags.Named(corso.NamedConfig)),
		fx.ResultTags(fxtags.Group(connectrpco.GroupHttpMiddleware)),
	),
)

// WithOtelConnectInterceptor provides an interceptor configured with otelconnect options.
var WithOtelConnectInterceptor = func(opts ...otelconnect.Option) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() ([]connect.ServerInterceptor, error) {
				interceptor, err := otelconnect.NewServerInterceptor(opts...)
				if err != nil {
					return nil, err
				}
				return []connect.ServerInterceptor{interceptor}, nil
			},
			fx.ResultTags(fxtags.Group(connectrpco.GroupConnectInterceptor)),
		),
	)
}

// SupplyConnectHandlerOptions supplies HTTP options for all registered RPC services.
func SupplyConnectHandlerOptions(opts ...connecthttp.Option) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connecthttp.Option {
				return opts
			},
			fx.ResultTags(fxtags.GroupFlatten(connectrpco.GroupConnectHandlerOptions)),
		),
	)
}

// SupplyConnectInterceptors supplies server interceptors for all registered RPC services.
// Order is preserved within one call; separate calls form unordered Fx groups.
func SupplyConnectInterceptors(interceptors ...connect.ServerInterceptor) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func() []connect.ServerInterceptor {
				return interceptors
			},
			fx.ResultTags(fxtags.Group(connectrpco.GroupConnectInterceptor)),
		),
	)
}

// SupplyConnectHandler registers a service before the server's HTTP routes are mounted.
func SupplyConnectHandler(register func(*connect.Server)) fx.Option {
	return fx.Supply(fx.Annotate(register, fx.ResultTags(fxtags.Group(connectrpco.GroupConnectHandler))))
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
