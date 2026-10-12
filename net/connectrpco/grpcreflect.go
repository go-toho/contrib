package connectrpco

import (
	"connectrpc.com/connect/v2"
	"connectrpc.com/grpcreflect/v2"
)

// RegisterGrpcReflect registers the gRPC reflection service, which describes
// all services registered on the server. For custom reflection options, call
// grpcreflect.Register directly.
func RegisterGrpcReflect(server *connect.Server) {
	grpcreflect.Register(server)
}
