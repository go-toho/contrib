package connectrpco

import (
	"slices"

	"connectrpc.com/connect/v2"
	"connectrpc.com/grpchealth/v2"
	"connectrpc.com/grpcreflect/v2"
)

// RegisterGrpcHealth registers the gRPC health service. It reports every
// service already registered on the server as serving, so pass it to
// RegisterUtilityServices to run it after the application services.
func RegisterGrpcHealth(server *connect.Server) {
	services := slices.DeleteFunc(serviceNames(server), func(name string) bool {
		// reflection is a utility service, not an application service
		return name == grpcreflect.ReflectV1ServiceName || name == grpcreflect.ReflectV1AlphaServiceName
	})
	if len(services) == 0 {
		return
	}

	grpchealth.Register(server, grpchealth.NewStaticChecker(services...))
}
