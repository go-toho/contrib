package connectrpco

import (
	"connectrpc.com/connect/v2"
	"connectrpc.com/grpcreflect/v2"
)

// NewGrpcReflectRegistrar creates a registrar for the gRPC reflection service.
// By default, reflection describes all services registered on the server.
func NewGrpcReflectRegistrar(options ...grpcreflect.Option) ServiceRegistrar {
	return func(server *connect.Server) {
		grpcreflect.Register(server, options...)
	}
}
