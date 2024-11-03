package connectrpco

import (
	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"

	"github.com/go-toho/contrib/net/httpo"
)

// NewGrpcHealthHandler creates a Connect gRPC health handler.
func NewGrpcHealthHandler(services []string, options ...connect.HandlerOption) []httpo.HttpPatternHandler {
	if len(services) == 0 {
		return nil
	}

	checker := grpchealth.NewStaticChecker(services...)

	pattern, handler := grpchealth.NewHandler(checker, options...)

	return []httpo.HttpPatternHandler{
		{Pattern: pattern, Handler: handler},
	}
}
