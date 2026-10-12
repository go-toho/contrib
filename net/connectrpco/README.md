# ConnectRPC v2

The ConnectRPC helpers use `connectrpc.com/connect/v2`,
`connectrpc.com/grpchealth/v2`, `connectrpc.com/grpcreflect/v2`, and
`connectrpc.com/otelconnect` v0.12. These upgrades must land together.

## Migrating callers

Regenerate your application's service code with the v2 Connect generator and
follow the [upstream migration guide](https://github.com/connectrpc/connect-go/blob/v2.0.0/docs/v2-migration.md)
for message signatures, metadata, errors, and custom interceptors. This repository
contains adapters, so protobuf generation runs in the application repository.

With Fx, replace the v1 handler factory:

```go
connectrpcofx.SupplyConnectHandler(func(options ...connect.HandlerOption) (string, http.Handler) {
    return userv1connect.NewUserServiceHandler(service, options...)
})
```

with a registration callback (`connect` now means `connectrpc.com/connect/v2`):

```go
connectrpcofx.SupplyConnectHandler(func(server *connect.Server) {
    userv1connect.RegisterUserServiceHandler(server, service)
})
```

For a service constructed by Fx, provide the callback through the same group:

```go
fx.Provide(fx.Annotate(
    func(service *UserService) func(*connect.Server) {
        return func(server *connect.Server) {
            userv1connect.RegisterUserServiceHandler(server, service)
        }
    },
    fx.ResultTags(`group:"connectrpco.ConnectHandler"`),
))
```

The module creates one `*connect.Server`, registers application services, adds
the enabled utility services, and then mounts all routes on its HTTP server.
Register services through these callbacks before mounting; a later `fx.Invoke`
that mutates the Connect server can run after HTTP routes have been installed.

Separate HTTP options from interceptors:

```go
connectrpcofx.SupplyConnectHandlerOptions(connecthttp.WithReadMaxBytes(8 << 20))
connectrpcofx.SupplyConnectInterceptors(authInterceptor, loggingInterceptor)
connectrpcofx.WithOtelConnectInterceptor(otelconnect.WithTracerProvider(provider))
```

`SupplyConnectHandlerOptions` now accepts `connecthttp.Option`.
`SupplyConnectInterceptors` accepts v2 `connect.ServerInterceptor` values;
interceptors apply to application, health, and reflection RPCs. Put interceptors
whose order matters in one call: order within that call is preserved, while
separate Fx groups (including the telemetry helper) are unordered.
`WithOtelConnectInterceptor` still accepts the same telemetry options and
propagates construction errors.

## Using the HTTP wrapper directly

Register services before passing the Connect server to the HTTP wrapper:

```go
rpcServer := connect.NewServer(interceptors...)
userv1connect.RegisterUserServiceHandler(rpcServer, service)
services := []string{userv1connect.UserServiceName}
connectrpco.RegisterGrpcHealthHandler(rpcServer, services)
connectrpco.RegisterGrpcReflectHandler(rpcServer, services)

server, err := connectrpco.NewConnectServer(
    connectrpco.WithConnectServer(rpcServer, connecthttp.WithReadMaxBytes(8 << 20)),
    connectrpco.HttpMiddleware(middleware...),
)
```

The existing HTTP handler, address, timeout, CORS, and authentication helpers
still work. `authn` remains HTTP middleware and retains an indirect Connect v1
dependency. Authentication callbacks should keep returning `authn.Errorf` or v1
Connect errors, because that middleware uses the v1 error writer.

## Exported API changes

| Previous API | v2 API |
| --- | --- |
| `SupplyConnectHandler(func(...connect.HandlerOption) (string, http.Handler))` | `SupplyConnectHandler(func(*connect.Server))` |
| `SupplyConnectHandlerOptions(...connect.HandlerOption)` | `SupplyConnectHandlerOptions(...connecthttp.Option)` and `SupplyConnectInterceptors(...connect.ServerInterceptor)` |
| `NewGrpcHealthHandler(services, options...)` | `RegisterGrpcHealthHandler(server, services)` |
| `NewGrpcReflectHandler(services, options...)` | `RegisterGrpcReflectHandler(server, services)` |
| `connectrpcofx.NewConnectServer(config, handlers, middleware, log, lifecycle) error` | `connectrpcofx.NewConnectServer(config, rpcServer, httpOptions, middleware) (*connectrpco.ConnectServer, error)` |
| `GroupConnectServiceNames` | Removed; application service names are collected from registered methods before utilities are added. |

`GroupConnectHandler` now contains `func(*connect.Server)` registrations.
`GroupUtilityConnectHandler` contains `func(*connect.Server, []string)` registrations.
`GroupConnectHandlerOptions` contains `connecthttp.Option` values.
The new `GroupConnectInterceptor` contains `[]connect.ServerInterceptor` chains.
Both the configured `*connect.Server` and `*connectrpco.ConnectServer` are available
from Fx; the latter is nil when the HTTP server is disabled. The module invokes
an internal function to attach the HTTP server's start and stop lifecycle hooks.

Health and reflection remain opt-in through `WithGrpcHealthHandler` and
`WithGrpcReflectHandler`. Both skip registration when there are no application
services. Reflection serves v1 and v1alpha and advertises application services
only, as before; health checks use the same application service list.

## Message sizes

The HTTP wrapper uses Connect v2's default read limit of 4 MiB per message.
Larger messages fail with `CodeResourceExhausted`. Set `connecthttp.WithReadMaxBytes`
through `WithConnectServer` or `SupplyConnectHandlerOptions` to override the limit;
pass `0` to explicitly allow unlimited reads. This changes v1's unlimited default.
V2 clients also default to 4 MiB, so configure their transport separately when
large responses are required.
