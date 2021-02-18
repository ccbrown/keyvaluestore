// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package client

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

// ServiceClient is the client API for Service service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServiceClient interface {
	ClearPartitions(ctx context.Context, in *ClearPartitionsRequest, opts ...grpc.CallOption) (*ClearPartitionsResponse, error)
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Set(ctx context.Context, in *SetRequest, opts ...grpc.CallOption) (*SetResponse, error)
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	Append(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendResponse, error)
	Filter(ctx context.Context, in *FilterRequest, opts ...grpc.CallOption) (*FilterResponse, error)
	MapSet(ctx context.Context, in *MapSetRequest, opts ...grpc.CallOption) (*MapSetResponse, error)
	MapDelete(ctx context.Context, in *MapDeleteRequest, opts ...grpc.CallOption) (*MapDeleteResponse, error)
	MapGetRange(ctx context.Context, in *MapGetRangeRequest, opts ...grpc.CallOption) (*MapGetRangeResponse, error)
	MapGetRangeByField(ctx context.Context, in *MapGetRangeByFieldRequest, opts ...grpc.CallOption) (*MapGetRangeByFieldResponse, error)
	MapCountRange(ctx context.Context, in *MapCountRangeRequest, opts ...grpc.CallOption) (*MapCountRangeResponse, error)
	MapCountRangeByField(ctx context.Context, in *MapCountRangeByFieldRequest, opts ...grpc.CallOption) (*MapCountRangeByFieldResponse, error)
}

type serviceClient struct {
	cc grpc.ClientConnInterface
}

func NewServiceClient(cc grpc.ClientConnInterface) ServiceClient {
	return &serviceClient{cc}
}

func (c *serviceClient) ClearPartitions(ctx context.Context, in *ClearPartitionsRequest, opts ...grpc.CallOption) (*ClearPartitionsResponse, error) {
	out := new(ClearPartitionsResponse)
	err := c.cc.Invoke(ctx, "/Service/clearPartitions", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, "/Service/get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Set(ctx context.Context, in *SetRequest, opts ...grpc.CallOption) (*SetResponse, error) {
	out := new(SetResponse)
	err := c.cc.Invoke(ctx, "/Service/set", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, "/Service/delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Append(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendResponse, error) {
	out := new(AppendResponse)
	err := c.cc.Invoke(ctx, "/Service/append", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Filter(ctx context.Context, in *FilterRequest, opts ...grpc.CallOption) (*FilterResponse, error) {
	out := new(FilterResponse)
	err := c.cc.Invoke(ctx, "/Service/filter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapSet(ctx context.Context, in *MapSetRequest, opts ...grpc.CallOption) (*MapSetResponse, error) {
	out := new(MapSetResponse)
	err := c.cc.Invoke(ctx, "/Service/map_set", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapDelete(ctx context.Context, in *MapDeleteRequest, opts ...grpc.CallOption) (*MapDeleteResponse, error) {
	out := new(MapDeleteResponse)
	err := c.cc.Invoke(ctx, "/Service/map_delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapGetRange(ctx context.Context, in *MapGetRangeRequest, opts ...grpc.CallOption) (*MapGetRangeResponse, error) {
	out := new(MapGetRangeResponse)
	err := c.cc.Invoke(ctx, "/Service/map_get_range", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapGetRangeByField(ctx context.Context, in *MapGetRangeByFieldRequest, opts ...grpc.CallOption) (*MapGetRangeByFieldResponse, error) {
	out := new(MapGetRangeByFieldResponse)
	err := c.cc.Invoke(ctx, "/Service/map_get_range_by_field", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapCountRange(ctx context.Context, in *MapCountRangeRequest, opts ...grpc.CallOption) (*MapCountRangeResponse, error) {
	out := new(MapCountRangeResponse)
	err := c.cc.Invoke(ctx, "/Service/map_count_range", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) MapCountRangeByField(ctx context.Context, in *MapCountRangeByFieldRequest, opts ...grpc.CallOption) (*MapCountRangeByFieldResponse, error) {
	out := new(MapCountRangeByFieldResponse)
	err := c.cc.Invoke(ctx, "/Service/map_count_range_by_field", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceServer is the server API for Service service.
// All implementations must embed UnimplementedServiceServer
// for forward compatibility
type ServiceServer interface {
	ClearPartitions(context.Context, *ClearPartitionsRequest) (*ClearPartitionsResponse, error)
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Set(context.Context, *SetRequest) (*SetResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	Append(context.Context, *AppendRequest) (*AppendResponse, error)
	Filter(context.Context, *FilterRequest) (*FilterResponse, error)
	MapSet(context.Context, *MapSetRequest) (*MapSetResponse, error)
	MapDelete(context.Context, *MapDeleteRequest) (*MapDeleteResponse, error)
	MapGetRange(context.Context, *MapGetRangeRequest) (*MapGetRangeResponse, error)
	MapGetRangeByField(context.Context, *MapGetRangeByFieldRequest) (*MapGetRangeByFieldResponse, error)
	MapCountRange(context.Context, *MapCountRangeRequest) (*MapCountRangeResponse, error)
	MapCountRangeByField(context.Context, *MapCountRangeByFieldRequest) (*MapCountRangeByFieldResponse, error)
	mustEmbedUnimplementedServiceServer()
}

// UnimplementedServiceServer must be embedded to have forward compatible implementations.
type UnimplementedServiceServer struct {
}

func (UnimplementedServiceServer) ClearPartitions(context.Context, *ClearPartitionsRequest) (*ClearPartitionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClearPartitions not implemented")
}
func (UnimplementedServiceServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedServiceServer) Set(context.Context, *SetRequest) (*SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Set not implemented")
}
func (UnimplementedServiceServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedServiceServer) Append(context.Context, *AppendRequest) (*AppendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Append not implemented")
}
func (UnimplementedServiceServer) Filter(context.Context, *FilterRequest) (*FilterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Filter not implemented")
}
func (UnimplementedServiceServer) MapSet(context.Context, *MapSetRequest) (*MapSetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapSet not implemented")
}
func (UnimplementedServiceServer) MapDelete(context.Context, *MapDeleteRequest) (*MapDeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapDelete not implemented")
}
func (UnimplementedServiceServer) MapGetRange(context.Context, *MapGetRangeRequest) (*MapGetRangeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapGetRange not implemented")
}
func (UnimplementedServiceServer) MapGetRangeByField(context.Context, *MapGetRangeByFieldRequest) (*MapGetRangeByFieldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapGetRangeByField not implemented")
}
func (UnimplementedServiceServer) MapCountRange(context.Context, *MapCountRangeRequest) (*MapCountRangeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapCountRange not implemented")
}
func (UnimplementedServiceServer) MapCountRangeByField(context.Context, *MapCountRangeByFieldRequest) (*MapCountRangeByFieldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MapCountRangeByField not implemented")
}
func (UnimplementedServiceServer) mustEmbedUnimplementedServiceServer() {}

// UnsafeServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServiceServer will
// result in compilation errors.
type UnsafeServiceServer interface {
	mustEmbedUnimplementedServiceServer()
}

func RegisterServiceServer(s grpc.ServiceRegistrar, srv ServiceServer) {
	s.RegisterService(&Service_ServiceDesc, srv)
}

func _Service_ClearPartitions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClearPartitionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).ClearPartitions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/clearPartitions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).ClearPartitions(ctx, req.(*ClearPartitionsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Set_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Set(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/set",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Set(ctx, req.(*SetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Append_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Append(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/append",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Append(ctx, req.(*AppendRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Filter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FilterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Filter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/filter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Filter(ctx, req.(*FilterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapSet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapSetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapSet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_set",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapSet(ctx, req.(*MapSetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapDelete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapDeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapDelete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapDelete(ctx, req.(*MapDeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapGetRange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapGetRangeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapGetRange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_get_range",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapGetRange(ctx, req.(*MapGetRangeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapGetRangeByField_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapGetRangeByFieldRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapGetRangeByField(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_get_range_by_field",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapGetRangeByField(ctx, req.(*MapGetRangeByFieldRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapCountRange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapCountRangeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapCountRange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_count_range",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapCountRange(ctx, req.(*MapCountRangeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_MapCountRangeByField_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MapCountRangeByFieldRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).MapCountRangeByField(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/map_count_range_by_field",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).MapCountRangeByField(ctx, req.(*MapCountRangeByFieldRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Service_ServiceDesc is the grpc.ServiceDesc for Service service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Service_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Service",
	HandlerType: (*ServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "clearPartitions",
			Handler:    _Service_ClearPartitions_Handler,
		},
		{
			MethodName: "get",
			Handler:    _Service_Get_Handler,
		},
		{
			MethodName: "set",
			Handler:    _Service_Set_Handler,
		},
		{
			MethodName: "delete",
			Handler:    _Service_Delete_Handler,
		},
		{
			MethodName: "append",
			Handler:    _Service_Append_Handler,
		},
		{
			MethodName: "filter",
			Handler:    _Service_Filter_Handler,
		},
		{
			MethodName: "map_set",
			Handler:    _Service_MapSet_Handler,
		},
		{
			MethodName: "map_delete",
			Handler:    _Service_MapDelete_Handler,
		},
		{
			MethodName: "map_get_range",
			Handler:    _Service_MapGetRange_Handler,
		},
		{
			MethodName: "map_get_range_by_field",
			Handler:    _Service_MapGetRangeByField_Handler,
		},
		{
			MethodName: "map_count_range",
			Handler:    _Service_MapCountRange_Handler,
		},
		{
			MethodName: "map_count_range_by_field",
			Handler:    _Service_MapCountRangeByField_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "client/client.proto",
}