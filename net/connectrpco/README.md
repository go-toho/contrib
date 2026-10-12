# connectrpco

ConnectRPC server helpers built on [connect-go v2](https://github.com/connectrpc/connect-go),
with an [fx](https://github.com/uber-go/fx) module in `connectrpcofx`.

The server owns a single `*connect.Server`. Services are added through
`RegisterFunc` functions, the server is mounted on an `http.ServeMux` with
`connecthttp.Mount`, and the result is served over h2c so gRPC clients work
without TLS.

## Usage with fx

```go
fx.New(
	connectrpcofx.Module,

	// provide the implementation, then register it with the generated function
	fx.Provide(fx.Annotate(NewPingServer, fx.As(new(pingv1connect.PingServiceHandler)))),
	connectrpcofx.ProvideServiceHandler(pingv1connect.RegisterPingServiceHandler),

	// utility services, registered after the application services
	connectrpcofx.WithGrpcHealthHandler,
	connectrpcofx.WithGrpcReflectHandler,

	// server-wide interceptors and HTTP handler options
	connectrpcofx.WithOtelConnectInterceptor(),
	connectrpcofx.SupplyServerInterceptors(validate.NewServerInterceptor()),
	connectrpcofx.SupplyConnectHttpOptions(connecthttp.WithReadMaxBytes(16<<20)),

	// plain HTTP handlers and middleware
	connectrpcofx.SupplyHttpHandlers(httpo.HttpPatternHandler{Pattern: "/healthz", Handler: healthz}),
	connectrpcofx.WithCorsHttpMiddleware,
)
```

Services supplied with `SupplyUtilityRegisterFuncs`, or provided into
`connectrpco.GroupUtilityRegisterFunc`, are registered after all application
services, so they can inspect them with `connect.Server.Specs`.
`WithGrpcHealthHandler` and `WithGrpcReflectHandler` use the same mechanism.

Interceptors are collected from an fx value group, which has no defined order.
If ordering matters, supply them together in one `SupplyServerInterceptors` call.

## Usage without fx

```go
srv, err := connectrpco.NewConnectServer(
	connectrpco.Address(":8080"),
	connectrpco.ServerInterceptors(interceptor),
	connectrpco.RegisterServices(
		func(s *connect.Server) { pingv1connect.RegisterPingServiceHandler(s, pingServer) },
	),
	// registered after all services, regardless of option order
	connectrpco.RegisterUtilityServices(
		connectrpco.RegisterGrpcHealth,
		connectrpco.RegisterGrpcReflect,
	),
)
```

## Testing

`connectrpcotest` starts a server with the same options as `NewConnectServer`
and returns clients for it. It uses HTTP/2 without TLS, so streaming RPCs work:

```go
srv := connectrpcotest.NewServer(t,
	connectrpco.ServerInterceptors(validate.NewServerInterceptor()),
	connectrpco.RegisterServices(func(s *connect.Server) {
		pingv1connect.RegisterPingServiceHandler(s, pingServer)
	}),
)
client := pingv1connect.NewPingServiceClient(srv.Client())
```

`srv.Client` accepts client interceptors. Use `srv.Transport(...)` for
`connecthttp` client options, or `srv.HttpClient()` for raw HTTP requests.

## Upgrading from the connect-go v1 version

Regenerate your code with the v2 `protoc-gen-connect-go` and run
[`connect-go-v2-migrate`](https://github.com/connectrpc/connect-go/blob/main/docs/v2-migration.md)
on your services first. Then update the wiring:

| v1 | v2 |
| --- | --- |
| `SupplyConnectHandler(pingv1connect.NewPingServiceHandler(...))` | `ProvideServiceHandler(pingv1connect.RegisterPingServiceHandler)` or `SupplyRegisterFuncs(...)` |
| `SupplyConnectHandlerOptions(connect.WithInterceptors(i))` | `SupplyServerInterceptors(i)` |
| `SupplyConnectHandlerOptions(connect.WithReadMaxBytes(n))` and other HTTP options | `SupplyConnectHttpOptions(connecthttp.WithReadMaxBytes(n))` |
| `GroupConnectHandler` (plain `httpo.HttpPatternHandler`) | `GroupHttpHandler` / `SupplyHttpHandlers(...)` |
| `GroupConnectHandlerOptions` | `GroupServerInterceptor` / `GroupConnectHttpOption` |
| `GroupUtilityConnectHandler` | `GroupUtilityRegisterFunc` / `SupplyUtilityRegisterFuncs(...)` |
| `GroupConnectServiceNames` | removed; service names are read from the server |
| `NewGrpcHealthHandler(services, opts...)` | `RegisterGrpcHealth` |
| `NewGrpcReflectHandler(services, opts...)` | `RegisterGrpcReflect`, or `grpcreflect.Register(server, opts...)` for options |

Behavior changes:

- **Read limit.** Messages are limited to 4 MiB by default (v1 was unbounded)
  and larger ones fail with `resource_exhausted`. Use
  `connecthttp.WithReadMaxBytes` to raise the limit, or pass `0` for no limit.
- **Interceptors are server-wide.** They apply to every service, including
  health and reflection. v1 allowed per-handler options.
- **Reflection** lists every registered service with a protobuf schema,
  including the reflection services themselves, instead of a static list.
- **Authentication** still uses `connectrpc.com/authn`, which has no Connect v2
  release yet. `NewAuthnMiddleware` accepts both `authn.Errorf` and v2
  `connect.NewError` errors from the authentication function. Error details on
  v2 errors are not forwarded.
