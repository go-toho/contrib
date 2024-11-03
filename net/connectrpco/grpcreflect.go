package connectrpco

import (
	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"

	"github.com/go-toho/contrib/net/httpo"
)

// NewGrpcReflectHandler creates a Connect gRPC reflection handler.
func NewGrpcReflectHandler(services []string, options ...connect.HandlerOption) []httpo.HttpPatternHandler {
	if len(services) == 0 {
		return nil
	}

	checker := grpcreflect.NewStaticReflector(services...)

	patternV1, handlerV1 := grpcreflect.NewHandlerV1(checker, options...)
	patternV1Alpha, handlerV1Alpha := grpcreflect.NewHandlerV1Alpha(checker, options...)

	return []httpo.HttpPatternHandler{
		{Pattern: patternV1, Handler: handlerV1},
		{Pattern: patternV1Alpha, Handler: handlerV1Alpha},
	}
}
