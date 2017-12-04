// Code generated by protoc-gen-go.
// source: proto/logservice.proto
// DO NOT EDIT!

/*
Package logservice is a generated protocol buffer package.

It is generated from these files:
	proto/logservice.proto

It has these top-level messages:
	LogAppDef
	LogLine
	LogRequest
	LogResponse
	LogFilter
	GetLogRequest
	LogEntry
	GetLogResponse
*/
package logservice

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

type LogAppDef struct {
	Status       string `protobuf:"bytes,1,opt,name=Status,json=status" json:"Status,omitempty"`
	Appname      string `protobuf:"bytes,2,opt,name=Appname,json=appname" json:"Appname,omitempty"`
	Repository   string `protobuf:"bytes,3,opt,name=Repository,json=repository" json:"Repository,omitempty"`
	Groupname    string `protobuf:"bytes,4,opt,name=Groupname,json=groupname" json:"Groupname,omitempty"`
	Namespace    string `protobuf:"bytes,5,opt,name=Namespace,json=namespace" json:"Namespace,omitempty"`
	DeploymentID string `protobuf:"bytes,6,opt,name=DeploymentID,json=deploymentID" json:"DeploymentID,omitempty"`
	StartupID    string `protobuf:"bytes,7,opt,name=StartupID,json=startupID" json:"StartupID,omitempty"`
}

func (m *LogAppDef) Reset()                    { *m = LogAppDef{} }
func (m *LogAppDef) String() string            { return proto.CompactTextString(m) }
func (*LogAppDef) ProtoMessage()               {}
func (*LogAppDef) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *LogAppDef) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func (m *LogAppDef) GetAppname() string {
	if m != nil {
		return m.Appname
	}
	return ""
}

func (m *LogAppDef) GetRepository() string {
	if m != nil {
		return m.Repository
	}
	return ""
}

func (m *LogAppDef) GetGroupname() string {
	if m != nil {
		return m.Groupname
	}
	return ""
}

func (m *LogAppDef) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *LogAppDef) GetDeploymentID() string {
	if m != nil {
		return m.DeploymentID
	}
	return ""
}

func (m *LogAppDef) GetStartupID() string {
	if m != nil {
		return m.StartupID
	}
	return ""
}

type LogLine struct {
	Time int64  `protobuf:"varint,1,opt,name=Time,json=time" json:"Time,omitempty"`
	Line string `protobuf:"bytes,2,opt,name=Line,json=line" json:"Line,omitempty"`
}

func (m *LogLine) Reset()                    { *m = LogLine{} }
func (m *LogLine) String() string            { return proto.CompactTextString(m) }
func (*LogLine) ProtoMessage()               {}
func (*LogLine) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *LogLine) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *LogLine) GetLine() string {
	if m != nil {
		return m.Line
	}
	return ""
}

type LogRequest struct {
	AppDef *LogAppDef `protobuf:"bytes,1,opt,name=AppDef,json=appDef" json:"AppDef,omitempty"`
	Lines  []*LogLine `protobuf:"bytes,2,rep,name=Lines,json=lines" json:"Lines,omitempty"`
}

func (m *LogRequest) Reset()                    { *m = LogRequest{} }
func (m *LogRequest) String() string            { return proto.CompactTextString(m) }
func (*LogRequest) ProtoMessage()               {}
func (*LogRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *LogRequest) GetAppDef() *LogAppDef {
	if m != nil {
		return m.AppDef
	}
	return nil
}

func (m *LogRequest) GetLines() []*LogLine {
	if m != nil {
		return m.Lines
	}
	return nil
}

type LogResponse struct {
}

func (m *LogResponse) Reset()                    { *m = LogResponse{} }
func (m *LogResponse) String() string            { return proto.CompactTextString(m) }
func (*LogResponse) ProtoMessage()               {}
func (*LogResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

type LogFilter struct {
	Host     string     `protobuf:"bytes,1,opt,name=Host,json=host" json:"Host,omitempty"`
	UserName string     `protobuf:"bytes,2,opt,name=UserName,json=userName" json:"UserName,omitempty"`
	AppDef   *LogAppDef `protobuf:"bytes,3,opt,name=AppDef,json=appDef" json:"AppDef,omitempty"`
}

func (m *LogFilter) Reset()                    { *m = LogFilter{} }
func (m *LogFilter) String() string            { return proto.CompactTextString(m) }
func (*LogFilter) ProtoMessage()               {}
func (*LogFilter) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *LogFilter) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *LogFilter) GetUserName() string {
	if m != nil {
		return m.UserName
	}
	return ""
}

func (m *LogFilter) GetAppDef() *LogAppDef {
	if m != nil {
		return m.AppDef
	}
	return nil
}

type GetLogRequest struct {
	// logical OR of stuff to retrieve - if null means EVERYTHING
	LogFilter []*LogFilter `protobuf:"bytes,1,rep,name=LogFilter,json=logFilter" json:"LogFilter,omitempty"`
	// minimum logid to retrieve (0=all) (negative means last n lines)
	MinimumLogID int64 `protobuf:"varint,2,opt,name=MinimumLogID,json=minimumLogID" json:"MinimumLogID,omitempty"`
}

func (m *GetLogRequest) Reset()                    { *m = GetLogRequest{} }
func (m *GetLogRequest) String() string            { return proto.CompactTextString(m) }
func (*GetLogRequest) ProtoMessage()               {}
func (*GetLogRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *GetLogRequest) GetLogFilter() []*LogFilter {
	if m != nil {
		return m.LogFilter
	}
	return nil
}

func (m *GetLogRequest) GetMinimumLogID() int64 {
	if m != nil {
		return m.MinimumLogID
	}
	return 0
}

type LogEntry struct {
	ID       uint64     `protobuf:"varint,1,opt,name=ID,json=iD" json:"ID,omitempty"`
	Host     string     `protobuf:"bytes,2,opt,name=Host,json=host" json:"Host,omitempty"`
	UserName string     `protobuf:"bytes,3,opt,name=UserName,json=userName" json:"UserName,omitempty"`
	Occured  uint64     `protobuf:"varint,4,opt,name=Occured,json=occured" json:"Occured,omitempty"`
	AppDef   *LogAppDef `protobuf:"bytes,5,opt,name=AppDef,json=appDef" json:"AppDef,omitempty"`
	Line     string     `protobuf:"bytes,6,opt,name=Line,json=line" json:"Line,omitempty"`
}

func (m *LogEntry) Reset()                    { *m = LogEntry{} }
func (m *LogEntry) String() string            { return proto.CompactTextString(m) }
func (*LogEntry) ProtoMessage()               {}
func (*LogEntry) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *LogEntry) GetID() uint64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *LogEntry) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *LogEntry) GetUserName() string {
	if m != nil {
		return m.UserName
	}
	return ""
}

func (m *LogEntry) GetOccured() uint64 {
	if m != nil {
		return m.Occured
	}
	return 0
}

func (m *LogEntry) GetAppDef() *LogAppDef {
	if m != nil {
		return m.AppDef
	}
	return nil
}

func (m *LogEntry) GetLine() string {
	if m != nil {
		return m.Line
	}
	return ""
}

type GetLogResponse struct {
	Entries []*LogEntry `protobuf:"bytes,1,rep,name=Entries,json=entries" json:"Entries,omitempty"`
}

func (m *GetLogResponse) Reset()                    { *m = GetLogResponse{} }
func (m *GetLogResponse) String() string            { return proto.CompactTextString(m) }
func (*GetLogResponse) ProtoMessage()               {}
func (*GetLogResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *GetLogResponse) GetEntries() []*LogEntry {
	if m != nil {
		return m.Entries
	}
	return nil
}

func init() {
	proto.RegisterType((*LogAppDef)(nil), "logservice.LogAppDef")
	proto.RegisterType((*LogLine)(nil), "logservice.LogLine")
	proto.RegisterType((*LogRequest)(nil), "logservice.LogRequest")
	proto.RegisterType((*LogResponse)(nil), "logservice.LogResponse")
	proto.RegisterType((*LogFilter)(nil), "logservice.LogFilter")
	proto.RegisterType((*GetLogRequest)(nil), "logservice.GetLogRequest")
	proto.RegisterType((*LogEntry)(nil), "logservice.LogEntry")
	proto.RegisterType((*GetLogResponse)(nil), "logservice.GetLogResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for LogService service

type LogServiceClient interface {
	LogCommandStdout(ctx context.Context, in *LogRequest, opts ...grpc.CallOption) (*LogResponse, error)
	GetLogCommandStdout(ctx context.Context, in *GetLogRequest, opts ...grpc.CallOption) (*GetLogResponse, error)
}

type logServiceClient struct {
	cc *grpc.ClientConn
}

func NewLogServiceClient(cc *grpc.ClientConn) LogServiceClient {
	return &logServiceClient{cc}
}

func (c *logServiceClient) LogCommandStdout(ctx context.Context, in *LogRequest, opts ...grpc.CallOption) (*LogResponse, error) {
	out := new(LogResponse)
	err := grpc.Invoke(ctx, "/logservice.LogService/LogCommandStdout", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *logServiceClient) GetLogCommandStdout(ctx context.Context, in *GetLogRequest, opts ...grpc.CallOption) (*GetLogResponse, error) {
	out := new(GetLogResponse)
	err := grpc.Invoke(ctx, "/logservice.LogService/GetLogCommandStdout", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for LogService service

type LogServiceServer interface {
	LogCommandStdout(context.Context, *LogRequest) (*LogResponse, error)
	GetLogCommandStdout(context.Context, *GetLogRequest) (*GetLogResponse, error)
}

func RegisterLogServiceServer(s *grpc.Server, srv LogServiceServer) {
	s.RegisterService(&_LogService_serviceDesc, srv)
}

func _LogService_LogCommandStdout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LogServiceServer).LogCommandStdout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logservice.LogService/LogCommandStdout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LogServiceServer).LogCommandStdout(ctx, req.(*LogRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LogService_GetLogCommandStdout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LogServiceServer).GetLogCommandStdout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logservice.LogService/GetLogCommandStdout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LogServiceServer).GetLogCommandStdout(ctx, req.(*GetLogRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _LogService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "logservice.LogService",
	HandlerType: (*LogServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "LogCommandStdout",
			Handler:    _LogService_LogCommandStdout_Handler,
		},
		{
			MethodName: "GetLogCommandStdout",
			Handler:    _LogService_GetLogCommandStdout_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/logservice.proto",
}

func init() { proto.RegisterFile("proto/logservice.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 529 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x8c, 0x53, 0xd1, 0x6e, 0xd3, 0x30,
	0x14, 0x55, 0x9a, 0x34, 0x69, 0x6f, 0xbb, 0x09, 0x79, 0x50, 0x42, 0x35, 0xa1, 0x29, 0x4f, 0xe3,
	0x81, 0x56, 0x74, 0x3f, 0xc0, 0xb4, 0xc0, 0xa8, 0x14, 0x86, 0x94, 0xc2, 0x07, 0x84, 0xf6, 0x2e,
	0x33, 0x4a, 0x7c, 0x4d, 0xec, 0x20, 0xf5, 0x6b, 0x78, 0xe4, 0xa3, 0xf8, 0x19, 0x14, 0x3b, 0xed,
	0xb2, 0x02, 0xd2, 0x1e, 0x7d, 0xce, 0xcd, 0xf1, 0x39, 0xf7, 0x38, 0x30, 0x91, 0x15, 0x69, 0x9a,
	0x17, 0x94, 0x2b, 0xac, 0x7e, 0xf0, 0x35, 0xce, 0x0c, 0xc0, 0xe0, 0x1e, 0x99, 0x9e, 0xe6, 0x44,
	0x79, 0x81, 0xf3, 0x4c, 0xf2, 0x79, 0x26, 0x04, 0xe9, 0x4c, 0x73, 0x12, 0xca, 0x4e, 0x46, 0xbf,
	0x1d, 0x18, 0x26, 0x94, 0x5f, 0x4a, 0x19, 0xe3, 0x2d, 0x9b, 0x80, 0xbf, 0xd2, 0x99, 0xae, 0x55,
	0xe8, 0x9c, 0x39, 0xe7, 0xc3, 0xd4, 0x57, 0xe6, 0xc4, 0x42, 0x08, 0x2e, 0xa5, 0x14, 0x59, 0x89,
	0x61, 0xcf, 0x10, 0x41, 0x66, 0x8f, 0xec, 0x25, 0x40, 0x8a, 0x92, 0x14, 0xd7, 0x54, 0x6d, 0x43,
	0xd7, 0x90, 0x50, 0xed, 0x11, 0x76, 0x0a, 0xc3, 0xeb, 0x8a, 0x6a, 0xfb, 0xad, 0x67, 0xe8, 0x61,
	0xbe, 0x03, 0x1a, 0xf6, 0x26, 0x2b, 0x51, 0xc9, 0x6c, 0x8d, 0x61, 0xdf, 0xb2, 0x62, 0x07, 0xb0,
	0x08, 0xc6, 0x31, 0xca, 0x82, 0xb6, 0x25, 0x0a, 0xbd, 0x8c, 0x43, 0xdf, 0x0c, 0x8c, 0x37, 0x1d,
	0xac, 0x51, 0x58, 0xe9, 0xac, 0xd2, 0xb5, 0x5c, 0xc6, 0x61, 0x60, 0x15, 0xd4, 0x0e, 0x88, 0xde,
	0x40, 0x90, 0x50, 0x9e, 0x70, 0x81, 0x8c, 0x81, 0xf7, 0x99, 0x97, 0x68, 0x82, 0xb9, 0xa9, 0xa7,
	0x79, 0x69, 0xb0, 0x86, 0x6b, 0x33, 0x79, 0x05, 0x17, 0x18, 0xdd, 0x02, 0x24, 0x94, 0xa7, 0xf8,
	0xbd, 0x46, 0xa5, 0xd9, 0x6b, 0xf0, 0xed, 0x6a, 0xcc, 0x77, 0xa3, 0xc5, 0xb3, 0x59, 0x67, 0xd7,
	0xfb, 0xbd, 0xa5, 0x7e, 0x66, 0xf7, 0xf7, 0x0a, 0xfa, 0x8d, 0xa0, 0x0a, 0x7b, 0x67, 0xee, 0xf9,
	0x68, 0x71, 0x72, 0x30, 0xdd, 0x70, 0x69, 0xbf, 0xb9, 0x46, 0x45, 0x47, 0x30, 0x32, 0xf7, 0x28,
	0x49, 0x42, 0x61, 0xf4, 0xcd, 0xd4, 0xf0, 0x9e, 0x17, 0x1a, 0xab, 0xc6, 0xd7, 0x07, 0x52, 0xba,
	0x2d, 0xc1, 0xbb, 0x23, 0xa5, 0xd9, 0x14, 0x06, 0x5f, 0x14, 0x56, 0x37, 0xf7, 0x1d, 0x0c, 0xea,
	0xf6, 0xdc, 0x71, 0xe9, 0x3e, 0xc2, 0x65, 0x74, 0x07, 0x47, 0xd7, 0xa8, 0x3b, 0x29, 0x2f, 0x3a,
	0x97, 0x87, 0x8e, 0xb1, 0x7e, 0x28, 0x61, 0xc9, 0x74, 0x58, 0xec, 0x4d, 0x46, 0x30, 0xfe, 0xc8,
	0x05, 0x2f, 0xeb, 0x32, 0xa1, 0x7c, 0x19, 0x1b, 0x53, 0x6e, 0x3a, 0x2e, 0x3b, 0x58, 0xf4, 0xcb,
	0x81, 0x41, 0x42, 0xf9, 0x3b, 0xa1, 0xab, 0x2d, 0x3b, 0x86, 0xde, 0x32, 0x36, 0x99, 0xbc, 0xb4,
	0xc7, 0xe3, 0x7d, 0xca, 0xde, 0x7f, 0x52, 0xba, 0x07, 0x29, 0x43, 0x08, 0x3e, 0xad, 0xd7, 0x75,
	0x85, 0x1b, 0xf3, 0x90, 0xbc, 0x34, 0x20, 0x7b, 0xec, 0xe4, 0xef, 0x3f, 0xa6, 0xa5, 0x5d, 0xed,
	0x7e, 0xa7, 0xf6, 0xb7, 0x70, 0xbc, 0xdb, 0x89, 0x6d, 0x84, 0xcd, 0x20, 0x68, 0x7c, 0x73, 0x54,
	0xed, 0x4a, 0x9e, 0x1e, 0xa8, 0x9a, 0x54, 0x69, 0x80, 0x76, 0x68, 0xf1, 0xd3, 0x31, 0x2f, 0x67,
	0x65, 0x07, 0xd8, 0x15, 0x3c, 0x49, 0x28, 0xbf, 0xa2, 0xb2, 0xcc, 0xc4, 0x66, 0xa5, 0x37, 0x54,
	0x6b, 0x36, 0x39, 0x50, 0x68, 0xf7, 0x3f, 0x7d, 0xfe, 0x17, 0xde, 0x7a, 0x48, 0xe0, 0xc4, 0xba,
	0x7a, 0xa8, 0xf3, 0xa2, 0x3b, 0xff, 0xa0, 0xca, 0xe9, 0xf4, 0x5f, 0x94, 0x55, 0xfb, 0xea, 0x9b,
	0x5f, 0xfe, 0xe2, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xca, 0xcb, 0xb5, 0x1e, 0x36, 0x04, 0x00,
	0x00,
}
