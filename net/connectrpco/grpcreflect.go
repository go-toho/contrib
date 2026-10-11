package connectrpco

import (
	"connectrpc.com/connect/v2"
	"connectrpc.com/grpcreflect/v2"
)

// RegisterGrpcReflectHandler registers both reflection APIs for the supplied services.
func RegisterGrpcReflectHandler(server *connect.Server, services []string) {
	if len(services) == 0 {
		return
	}
	grpcreflect.Register(server, grpcreflect.WithNamer(grpcreflect.NamerFunc(func() []string {
		return services
	})))
}
