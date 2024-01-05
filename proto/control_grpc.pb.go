// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.1
// source: control.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ControlPlaneClient is the client API for ControlPlane service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ControlPlaneClient interface {
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	Updater(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (ControlPlane_UpdaterClient, error)
	Punch(ctx context.Context, in *PunchRequest, opts ...grpc.CallOption) (*PunchReply, error)
}

type controlPlaneClient struct {
	cc grpc.ClientConnInterface
}

func NewControlPlaneClient(cc grpc.ClientConnInterface) ControlPlaneClient {
	return &controlPlaneClient{cc}
}

func (c *controlPlaneClient) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlPlaneClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlPlaneClient) Updater(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (ControlPlane_UpdaterClient, error) {
	stream, err := c.cc.NewStream(ctx, &ControlPlane_ServiceDesc.Streams[0], "/proto.ControlPlane/Updater", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlPlaneUpdaterClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ControlPlane_UpdaterClient interface {
	Recv() (*Update, error)
	grpc.ClientStream
}

type controlPlaneUpdaterClient struct {
	grpc.ClientStream
}

func (x *controlPlaneUpdaterClient) Recv() (*Update, error) {
	m := new(Update)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *controlPlaneClient) Punch(ctx context.Context, in *PunchRequest, opts ...grpc.CallOption) (*PunchReply, error) {
	out := new(PunchReply)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/Punch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ControlPlaneServer is the server API for ControlPlane service.
// All implementations must embed UnimplementedControlPlaneServer
// for forward compatibility
type ControlPlaneServer interface {
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	Updater(*UpdateRequest, ControlPlane_UpdaterServer) error
	Punch(context.Context, *PunchRequest) (*PunchReply, error)
	mustEmbedUnimplementedControlPlaneServer()
}

// UnimplementedControlPlaneServer must be embedded to have forward compatible implementations.
type UnimplementedControlPlaneServer struct {
}

func (UnimplementedControlPlaneServer) Login(context.Context, *LoginRequest) (*LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedControlPlaneServer) Register(context.Context, *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedControlPlaneServer) Updater(*UpdateRequest, ControlPlane_UpdaterServer) error {
	return status.Errorf(codes.Unimplemented, "method Updater not implemented")
}
func (UnimplementedControlPlaneServer) Punch(context.Context, *PunchRequest) (*PunchReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Punch not implemented")
}
func (UnimplementedControlPlaneServer) mustEmbedUnimplementedControlPlaneServer() {}

// UnsafeControlPlaneServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ControlPlaneServer will
// result in compilation errors.
type UnsafeControlPlaneServer interface {
	mustEmbedUnimplementedControlPlaneServer()
}

func RegisterControlPlaneServer(s grpc.ServiceRegistrar, srv ControlPlaneServer) {
	s.RegisterService(&ControlPlane_ServiceDesc, srv)
}

func _ControlPlane_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/Login",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).Login(ctx, req.(*LoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlPlane_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlPlane_Updater_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(UpdateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlPlaneServer).Updater(m, &controlPlaneUpdaterServer{stream})
}

type ControlPlane_UpdaterServer interface {
	Send(*Update) error
	grpc.ServerStream
}

type controlPlaneUpdaterServer struct {
	grpc.ServerStream
}

func (x *controlPlaneUpdaterServer) Send(m *Update) error {
	return x.ServerStream.SendMsg(m)
}

func _ControlPlane_Punch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PunchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).Punch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/Punch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).Punch(ctx, req.(*PunchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ControlPlane_ServiceDesc is the grpc.ServiceDesc for ControlPlane service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ControlPlane_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.ControlPlane",
	HandlerType: (*ControlPlaneServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Login",
			Handler:    _ControlPlane_Login_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _ControlPlane_Register_Handler,
		},
		{
			MethodName: "Punch",
			Handler:    _ControlPlane_Punch_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Updater",
			Handler:       _ControlPlane_Updater_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "control.proto",
}
