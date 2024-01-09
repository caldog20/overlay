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
	LoginPeer(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	RegisterPeer(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	SetPeerEndpoint(ctx context.Context, in *Endpoint, opts ...grpc.CallOption) (*EmptyResponse, error)
	Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (ControlPlane_UpdateClient, error)
	Punch(ctx context.Context, in *PunchRequest, opts ...grpc.CallOption) (*EmptyResponse, error)
}

type controlPlaneClient struct {
	cc grpc.ClientConnInterface
}

func NewControlPlaneClient(cc grpc.ClientConnInterface) ControlPlaneClient {
	return &controlPlaneClient{cc}
}

func (c *controlPlaneClient) LoginPeer(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/LoginPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlPlaneClient) RegisterPeer(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/RegisterPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlPlaneClient) SetPeerEndpoint(ctx context.Context, in *Endpoint, opts ...grpc.CallOption) (*EmptyResponse, error) {
	out := new(EmptyResponse)
	err := c.cc.Invoke(ctx, "/proto.ControlPlane/SetPeerEndpoint", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlPlaneClient) Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (ControlPlane_UpdateClient, error) {
	stream, err := c.cc.NewStream(ctx, &ControlPlane_ServiceDesc.Streams[0], "/proto.ControlPlane/Update", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlPlaneUpdateClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ControlPlane_UpdateClient interface {
	Recv() (*UpdateResponse, error)
	grpc.ClientStream
}

type controlPlaneUpdateClient struct {
	grpc.ClientStream
}

func (x *controlPlaneUpdateClient) Recv() (*UpdateResponse, error) {
	m := new(UpdateResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *controlPlaneClient) Punch(ctx context.Context, in *PunchRequest, opts ...grpc.CallOption) (*EmptyResponse, error) {
	out := new(EmptyResponse)
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
	LoginPeer(context.Context, *LoginRequest) (*LoginResponse, error)
	RegisterPeer(context.Context, *RegisterRequest) (*RegisterResponse, error)
	SetPeerEndpoint(context.Context, *Endpoint) (*EmptyResponse, error)
	Update(*UpdateRequest, ControlPlane_UpdateServer) error
	Punch(context.Context, *PunchRequest) (*EmptyResponse, error)
	mustEmbedUnimplementedControlPlaneServer()
}

// UnimplementedControlPlaneServer must be embedded to have forward compatible implementations.
type UnimplementedControlPlaneServer struct {
}

func (UnimplementedControlPlaneServer) LoginPeer(context.Context, *LoginRequest) (*LoginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoginPeer not implemented")
}
func (UnimplementedControlPlaneServer) RegisterPeer(context.Context, *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterPeer not implemented")
}
func (UnimplementedControlPlaneServer) SetPeerEndpoint(context.Context, *Endpoint) (*EmptyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetPeerEndpoint not implemented")
}
func (UnimplementedControlPlaneServer) Update(*UpdateRequest, ControlPlane_UpdateServer) error {
	return status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedControlPlaneServer) Punch(context.Context, *PunchRequest) (*EmptyResponse, error) {
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

func _ControlPlane_LoginPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).LoginPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/LoginPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).LoginPeer(ctx, req.(*LoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlPlane_RegisterPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).RegisterPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/RegisterPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).RegisterPeer(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlPlane_SetPeerEndpoint_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Endpoint)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlPlaneServer).SetPeerEndpoint(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.ControlPlane/SetPeerEndpoint",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlPlaneServer).SetPeerEndpoint(ctx, req.(*Endpoint))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlPlane_Update_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(UpdateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlPlaneServer).Update(m, &controlPlaneUpdateServer{stream})
}

type ControlPlane_UpdateServer interface {
	Send(*UpdateResponse) error
	grpc.ServerStream
}

type controlPlaneUpdateServer struct {
	grpc.ServerStream
}

func (x *controlPlaneUpdateServer) Send(m *UpdateResponse) error {
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
			MethodName: "LoginPeer",
			Handler:    _ControlPlane_LoginPeer_Handler,
		},
		{
			MethodName: "RegisterPeer",
			Handler:    _ControlPlane_RegisterPeer_Handler,
		},
		{
			MethodName: "SetPeerEndpoint",
			Handler:    _ControlPlane_SetPeerEndpoint_Handler,
		},
		{
			MethodName: "Punch",
			Handler:    _ControlPlane_Punch_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Update",
			Handler:       _ControlPlane_Update_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "control.proto",
}
