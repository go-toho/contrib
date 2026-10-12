// Package echotest provides a minimal Connect service for tests.
package echotest

import (
	"context"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceName = "test.v1.EchoService"
	Procedure   = "/" + ServiceName + "/Echo"
)

// echoMethod describes the Echo method, the way generated code does, so that
// reflection can list the service.
var echoMethod = func() protoreflect.MethodDescriptor {
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/v1/echo.proto"),
		Package:    proto.String("test.v1"),
		Dependency: []string{"google/protobuf/wrappers.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("EchoService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       proto.String("Echo"),
				InputType:  proto.String(".google.protobuf.StringValue"),
				OutputType: proto.String(".google.protobuf.StringValue"),
			}},
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	return file.Services().Get(0).Methods().Get(0)
}()

// Register registers a unary service that echoes a StringValue.
func Register(server *connect.Server) {
	server.Register(connect.Method{
		Spec: connect.Spec{
			StreamType: connect.StreamTypeUnary,
			Procedure:  Procedure,
			Schema:     echoMethod,
		},
		Handler: func(_ context.Context, _ connect.Spec, stream connect.ServerStream) error {
			var msg wrapperspb.StringValue
			if err := stream.Receive(&msg); err != nil {
				return err
			}
			return stream.Send(&msg)
		},
	})
}
