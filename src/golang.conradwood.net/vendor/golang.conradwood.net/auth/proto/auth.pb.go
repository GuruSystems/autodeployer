// Code generated by protoc-gen-go.
// source: proto/auth.proto
// DO NOT EDIT!

/*
Package auth is a generated protocol buffer package.

It is generated from these files:
	proto/auth.proto

It has these top-level messages:
	VerifyRequest
	VerifyResponse
	GetDetailRequest
	GetDetailResponse
	AuthenticatePasswordRequest
	VerifyPasswordResponse
*/
package auth

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

//
// import "google/protobuf/empty.proto";
// import "google/protobuf/duration.proto";
// import "examples/sub/message.proto";
// import "examples/sub2/message.proto";
// import "google/protobuf/timestamp.proto";
type VerifyRequest struct {
	Token string `protobuf:"bytes,1,opt,name=Token,json=token" json:"Token,omitempty"`
}

func (m *VerifyRequest) Reset()                    { *m = VerifyRequest{} }
func (m *VerifyRequest) String() string            { return proto.CompactTextString(m) }
func (*VerifyRequest) ProtoMessage()               {}
func (*VerifyRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *VerifyRequest) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

type VerifyResponse struct {
	UserID string `protobuf:"bytes,1,opt,name=UserID,json=userID" json:"UserID,omitempty"`
}

func (m *VerifyResponse) Reset()                    { *m = VerifyResponse{} }
func (m *VerifyResponse) String() string            { return proto.CompactTextString(m) }
func (*VerifyResponse) ProtoMessage()               {}
func (*VerifyResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *VerifyResponse) GetUserID() string {
	if m != nil {
		return m.UserID
	}
	return ""
}

type GetDetailRequest struct {
	UserID string `protobuf:"bytes,1,opt,name=UserID,json=userID" json:"UserID,omitempty"`
}

func (m *GetDetailRequest) Reset()                    { *m = GetDetailRequest{} }
func (m *GetDetailRequest) String() string            { return proto.CompactTextString(m) }
func (*GetDetailRequest) ProtoMessage()               {}
func (*GetDetailRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *GetDetailRequest) GetUserID() string {
	if m != nil {
		return m.UserID
	}
	return ""
}

type GetDetailResponse struct {
	UserID    string `protobuf:"bytes,1,opt,name=UserID,json=userID" json:"UserID,omitempty"`
	Email     string `protobuf:"bytes,2,opt,name=Email,json=email" json:"Email,omitempty"`
	FirstName string `protobuf:"bytes,3,opt,name=FirstName,json=firstName" json:"FirstName,omitempty"`
	LastName  string `protobuf:"bytes,4,opt,name=LastName,json=lastName" json:"LastName,omitempty"`
}

func (m *GetDetailResponse) Reset()                    { *m = GetDetailResponse{} }
func (m *GetDetailResponse) String() string            { return proto.CompactTextString(m) }
func (*GetDetailResponse) ProtoMessage()               {}
func (*GetDetailResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *GetDetailResponse) GetUserID() string {
	if m != nil {
		return m.UserID
	}
	return ""
}

func (m *GetDetailResponse) GetEmail() string {
	if m != nil {
		return m.Email
	}
	return ""
}

func (m *GetDetailResponse) GetFirstName() string {
	if m != nil {
		return m.FirstName
	}
	return ""
}

func (m *GetDetailResponse) GetLastName() string {
	if m != nil {
		return m.LastName
	}
	return ""
}

type AuthenticatePasswordRequest struct {
	Email    string `protobuf:"bytes,1,opt,name=Email,json=email" json:"Email,omitempty"`
	Password string `protobuf:"bytes,2,opt,name=Password,json=password" json:"Password,omitempty"`
}

func (m *AuthenticatePasswordRequest) Reset()                    { *m = AuthenticatePasswordRequest{} }
func (m *AuthenticatePasswordRequest) String() string            { return proto.CompactTextString(m) }
func (*AuthenticatePasswordRequest) ProtoMessage()               {}
func (*AuthenticatePasswordRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *AuthenticatePasswordRequest) GetEmail() string {
	if m != nil {
		return m.Email
	}
	return ""
}

func (m *AuthenticatePasswordRequest) GetPassword() string {
	if m != nil {
		return m.Password
	}
	return ""
}

type VerifyPasswordResponse struct {
	User  *GetDetailResponse `protobuf:"bytes,1,opt,name=User,json=user" json:"User,omitempty"`
	Token string             `protobuf:"bytes,2,opt,name=Token,json=token" json:"Token,omitempty"`
}

func (m *VerifyPasswordResponse) Reset()                    { *m = VerifyPasswordResponse{} }
func (m *VerifyPasswordResponse) String() string            { return proto.CompactTextString(m) }
func (*VerifyPasswordResponse) ProtoMessage()               {}
func (*VerifyPasswordResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *VerifyPasswordResponse) GetUser() *GetDetailResponse {
	if m != nil {
		return m.User
	}
	return nil
}

func (m *VerifyPasswordResponse) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

func init() {
	proto.RegisterType((*VerifyRequest)(nil), "auth.VerifyRequest")
	proto.RegisterType((*VerifyResponse)(nil), "auth.VerifyResponse")
	proto.RegisterType((*GetDetailRequest)(nil), "auth.GetDetailRequest")
	proto.RegisterType((*GetDetailResponse)(nil), "auth.GetDetailResponse")
	proto.RegisterType((*AuthenticatePasswordRequest)(nil), "auth.AuthenticatePasswordRequest")
	proto.RegisterType((*VerifyPasswordResponse)(nil), "auth.VerifyPasswordResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for AuthenticationService service

type AuthenticationServiceClient interface {
	// authenticate a user by username/password, return token
	AuthenticatePassword(ctx context.Context, in *AuthenticatePasswordRequest, opts ...grpc.CallOption) (*VerifyPasswordResponse, error)
	// verify a user by token
	VerifyUserToken(ctx context.Context, in *VerifyRequest, opts ...grpc.CallOption) (*VerifyResponse, error)
	GetUserDetail(ctx context.Context, in *GetDetailRequest, opts ...grpc.CallOption) (*GetDetailResponse, error)
}

type authenticationServiceClient struct {
	cc *grpc.ClientConn
}

func NewAuthenticationServiceClient(cc *grpc.ClientConn) AuthenticationServiceClient {
	return &authenticationServiceClient{cc}
}

func (c *authenticationServiceClient) AuthenticatePassword(ctx context.Context, in *AuthenticatePasswordRequest, opts ...grpc.CallOption) (*VerifyPasswordResponse, error) {
	out := new(VerifyPasswordResponse)
	err := grpc.Invoke(ctx, "/auth.AuthenticationService/AuthenticatePassword", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authenticationServiceClient) VerifyUserToken(ctx context.Context, in *VerifyRequest, opts ...grpc.CallOption) (*VerifyResponse, error) {
	out := new(VerifyResponse)
	err := grpc.Invoke(ctx, "/auth.AuthenticationService/VerifyUserToken", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authenticationServiceClient) GetUserDetail(ctx context.Context, in *GetDetailRequest, opts ...grpc.CallOption) (*GetDetailResponse, error) {
	out := new(GetDetailResponse)
	err := grpc.Invoke(ctx, "/auth.AuthenticationService/GetUserDetail", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for AuthenticationService service

type AuthenticationServiceServer interface {
	// authenticate a user by username/password, return token
	AuthenticatePassword(context.Context, *AuthenticatePasswordRequest) (*VerifyPasswordResponse, error)
	// verify a user by token
	VerifyUserToken(context.Context, *VerifyRequest) (*VerifyResponse, error)
	GetUserDetail(context.Context, *GetDetailRequest) (*GetDetailResponse, error)
}

func RegisterAuthenticationServiceServer(s *grpc.Server, srv AuthenticationServiceServer) {
	s.RegisterService(&_AuthenticationService_serviceDesc, srv)
}

func _AuthenticationService_AuthenticatePassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthenticatePasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthenticationServiceServer).AuthenticatePassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/auth.AuthenticationService/AuthenticatePassword",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthenticationServiceServer).AuthenticatePassword(ctx, req.(*AuthenticatePasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthenticationService_VerifyUserToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthenticationServiceServer).VerifyUserToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/auth.AuthenticationService/VerifyUserToken",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthenticationServiceServer).VerifyUserToken(ctx, req.(*VerifyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthenticationService_GetUserDetail_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDetailRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthenticationServiceServer).GetUserDetail(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/auth.AuthenticationService/GetUserDetail",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthenticationServiceServer).GetUserDetail(ctx, req.(*GetDetailRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AuthenticationService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "auth.AuthenticationService",
	HandlerType: (*AuthenticationServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AuthenticatePassword",
			Handler:    _AuthenticationService_AuthenticatePassword_Handler,
		},
		{
			MethodName: "VerifyUserToken",
			Handler:    _AuthenticationService_VerifyUserToken_Handler,
		},
		{
			MethodName: "GetUserDetail",
			Handler:    _AuthenticationService_GetUserDetail_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/auth.proto",
}

func init() { proto.RegisterFile("proto/auth.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 361 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x52, 0x5d, 0x4b, 0xeb, 0x40,
	0x10, 0x6d, 0x7a, 0xd3, 0x92, 0xce, 0xa5, 0xf7, 0xf6, 0xee, 0xad, 0xb5, 0xc4, 0x3e, 0x68, 0x40,
	0x28, 0x0a, 0x2d, 0xd4, 0x77, 0x41, 0xa9, 0x16, 0x41, 0x54, 0xea, 0x07, 0x88, 0x4f, 0x6b, 0x9d,
	0xb6, 0x8b, 0xe9, 0x6e, 0xcc, 0x6e, 0x14, 0xf1, 0x7f, 0xfb, 0x2c, 0xbb, 0x9b, 0xd8, 0x48, 0x3f,
	0xde, 0x72, 0x76, 0xce, 0x9c, 0x39, 0x67, 0x26, 0x50, 0x8b, 0x62, 0xa1, 0x44, 0x97, 0x26, 0x6a,
	0xda, 0x31, 0x9f, 0xc4, 0xd5, 0xdf, 0x7e, 0x6b, 0x22, 0xc4, 0x24, 0xc4, 0x2e, 0x8d, 0x58, 0x97,
	0x72, 0x2e, 0x14, 0x55, 0x4c, 0x70, 0x69, 0x39, 0xc1, 0x2e, 0x54, 0xef, 0x30, 0x66, 0xe3, 0xf7,
	0x21, 0xbe, 0x24, 0x28, 0x15, 0xa9, 0x43, 0xe9, 0x46, 0x3c, 0x23, 0x6f, 0x3a, 0xdb, 0x4e, 0xbb,
	0x32, 0x2c, 0x29, 0x0d, 0x82, 0x36, 0xfc, 0xc9, 0x68, 0x32, 0x12, 0x5c, 0x22, 0x69, 0x40, 0xf9,
	0x56, 0x62, 0x7c, 0xd6, 0x4f, 0x89, 0xe5, 0xc4, 0xa0, 0x60, 0x0f, 0x6a, 0x03, 0x54, 0x7d, 0x54,
	0x94, 0x85, 0x99, 0xe6, 0x2a, 0xee, 0x07, 0xfc, 0xcb, 0x71, 0xd7, 0x0b, 0x6b, 0x63, 0x27, 0x33,
	0xca, 0xc2, 0x66, 0xd1, 0x1a, 0x43, 0x0d, 0x48, 0x0b, 0x2a, 0xa7, 0x2c, 0x96, 0xea, 0x82, 0xce,
	0xb0, 0xf9, 0xcb, 0x54, 0x2a, 0xe3, 0xec, 0x81, 0xf8, 0xe0, 0x9d, 0xd3, 0xb4, 0xe8, 0x9a, 0xa2,
	0x17, 0xa6, 0x38, 0xb8, 0x84, 0xad, 0xa3, 0x44, 0x4d, 0x91, 0x2b, 0x36, 0xa2, 0x0a, 0xaf, 0xa8,
	0x94, 0x6f, 0x22, 0x7e, 0xca, 0xed, 0xc1, 0x8e, 0x73, 0xf2, 0xe3, 0x7c, 0xf0, 0x32, 0x62, 0xea,
	0xc3, 0x8b, 0x52, 0x1c, 0x3c, 0x40, 0xc3, 0xee, 0x68, 0x2e, 0x95, 0x46, 0xda, 0x07, 0x57, 0x47,
	0x32, 0x52, 0xbf, 0x7b, 0x9b, 0x1d, 0x73, 0xa3, 0x85, 0xe4, 0x43, 0x57, 0x27, 0x9d, 0x1f, 0xa0,
	0x98, 0x3b, 0x40, 0xef, 0xd3, 0x81, 0x8d, 0x9c, 0x5d, 0x26, 0xf8, 0x35, 0xc6, 0xaf, 0x6c, 0x84,
	0xe4, 0x1e, 0xea, 0xcb, 0x72, 0x90, 0x1d, 0x3b, 0x66, 0x4d, 0x46, 0xbf, 0x65, 0x29, 0xcb, 0x5d,
	0x07, 0x05, 0x72, 0x08, 0x7f, 0x6d, 0x4d, 0xbb, 0x37, 0xa6, 0xc8, 0xff, 0x7c, 0x4b, 0xa6, 0x53,
	0xff, 0xf9, 0xf8, 0xdd, 0x7f, 0x0c, 0xd5, 0x01, 0x2a, 0xdd, 0x6c, 0x93, 0x92, 0xc6, 0x42, 0x74,
	0x2b, 0xb0, 0x6a, 0x25, 0x41, 0xe1, 0xb1, 0x6c, 0xfe, 0xd3, 0x83, 0xaf, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xdf, 0xcd, 0x77, 0x2a, 0xdf, 0x02, 0x00, 0x00,
}
