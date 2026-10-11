package connectrpco

import (
	"connectrpc.com/connect/v2"
	"connectrpc.com/grpchealth/v2"
)

// RegisterGrpcHealthHandler registers health checks for the supplied services.
func RegisterGrpcHealthHandler(server *connect.Server, services []string) {
	if len(services) == 0 {
		return
	}
	grpchealth.Register(server, grpchealth.NewStaticChecker(services...))
}
