// Code generated by protoc-gen-go.
// source: dlock.proto
// DO NOT EDIT!

/*
Package dlock is a generated protocol buffer package.

It is generated from these files:
	dlock.proto

It has these top-level messages:
	Request
	Response
	RequestLock
*/
package dlock

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RequestType int32

const (
	RequestType_Invalid RequestType = 0
	RequestType_Ping    RequestType = 1
	RequestType_Lock    RequestType = 2
	RequestType_Unlock  RequestType = 3
)

var RequestType_name = map[int32]string{
	0: "Invalid",
	1: "Ping",
	2: "Lock",
	3: "Unlock",
}
var RequestType_value = map[string]int32{
	"Invalid": 0,
	"Ping":    1,
	"Lock":    2,
	"Unlock":  3,
}

func (x RequestType) String() string {
	return proto.EnumName(RequestType_name, int32(x))
}
func (RequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type ResponseStatus int32

const (
	// Error codes:
	// 1-99: protocol level errors
	// 100-119: [lock] input validation errors
	// 120-139: [lock] response errors for valid input
	ResponseStatus_Ok          ResponseStatus = 0
	ResponseStatus_General     ResponseStatus = 1
	ResponseStatus_Version     ResponseStatus = 2
	ResponseStatus_InvalidType ResponseStatus = 3
	// Lock 100-199
	ResponseStatus_TooManyKeys    ResponseStatus = 100
	ResponseStatus_AcquireTimeout ResponseStatus = 120
)

var ResponseStatus_name = map[int32]string{
	0:   "Ok",
	1:   "General",
	2:   "Version",
	3:   "InvalidType",
	100: "TooManyKeys",
	120: "AcquireTimeout",
}
var ResponseStatus_value = map[string]int32{
	"Ok":             0,
	"General":        1,
	"Version":        2,
	"InvalidType":    3,
	"TooManyKeys":    100,
	"AcquireTimeout": 120,
}

func (x ResponseStatus) String() string {
	return proto.EnumName(ResponseStatus_name, int32(x))
}
func (ResponseStatus) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type Request struct {
	Version     uint32      `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Id          uint64      `protobuf:"varint,2,opt,name=id" json:"id,omitempty"`
	AccessToken string      `protobuf:"bytes,3,opt,name=access_token,json=accessToken" json:"access_token,omitempty"`
	Type        RequestType `protobuf:"varint,4,opt,name=type,enum=dlock.RequestType" json:"type,omitempty"`
	// Ping is empty
	Lock *RequestLock `protobuf:"bytes,51,opt,name=lock" json:"lock,omitempty"`
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Request) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Request) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Request) GetAccessToken() string {
	if m != nil {
		return m.AccessToken
	}
	return ""
}

func (m *Request) GetType() RequestType {
	if m != nil {
		return m.Type
	}
	return RequestType_Invalid
}

func (m *Request) GetLock() *RequestLock {
	if m != nil {
		return m.Lock
	}
	return nil
}

type Response struct {
	Version        uint32         `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	RequestId      uint64         `protobuf:"varint,2,opt,name=request_id,json=requestId" json:"request_id,omitempty"`
	Status         ResponseStatus `protobuf:"varint,3,opt,name=status,enum=dlock.ResponseStatus" json:"status,omitempty"`
	ErrorText      string         `protobuf:"bytes,4,opt,name=error_text,json=errorText" json:"error_text,omitempty"`
	Keys           []string       `protobuf:"bytes,5,rep,name=keys" json:"keys,omitempty"`
	ServerUnixTime int64          `protobuf:"varint,6,opt,name=server_unix_time,json=serverUnixTime" json:"server_unix_time,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Response) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Response) GetRequestId() uint64 {
	if m != nil {
		return m.RequestId
	}
	return 0
}

func (m *Response) GetStatus() ResponseStatus {
	if m != nil {
		return m.Status
	}
	return ResponseStatus_Ok
}

func (m *Response) GetErrorText() string {
	if m != nil {
		return m.ErrorText
	}
	return ""
}

func (m *Response) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

func (m *Response) GetServerUnixTime() int64 {
	if m != nil {
		return m.ServerUnixTime
	}
	return 0
}

type RequestLock struct {
	WaitMicro    uint64   `protobuf:"varint,1,opt,name=wait_micro,json=waitMicro" json:"wait_micro,omitempty"`
	ReleaseMicro uint64   `protobuf:"varint,2,opt,name=release_micro,json=releaseMicro" json:"release_micro,omitempty"`
	Keys         []string `protobuf:"bytes,3,rep,name=keys" json:"keys,omitempty"`
}

func (m *RequestLock) Reset()                    { *m = RequestLock{} }
func (m *RequestLock) String() string            { return proto.CompactTextString(m) }
func (*RequestLock) ProtoMessage()               {}
func (*RequestLock) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *RequestLock) GetWaitMicro() uint64 {
	if m != nil {
		return m.WaitMicro
	}
	return 0
}

func (m *RequestLock) GetReleaseMicro() uint64 {
	if m != nil {
		return m.ReleaseMicro
	}
	return 0
}

func (m *RequestLock) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

func init() {
	proto.RegisterType((*Request)(nil), "dlock.Request")
	proto.RegisterType((*Response)(nil), "dlock.Response")
	proto.RegisterType((*RequestLock)(nil), "dlock.RequestLock")
	proto.RegisterEnum("dlock.RequestType", RequestType_name, RequestType_value)
	proto.RegisterEnum("dlock.ResponseStatus", ResponseStatus_name, ResponseStatus_value)
}

func init() { proto.RegisterFile("dlock.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 428 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x92, 0xcf, 0x6e, 0xd4, 0x30,
	0x10, 0xc6, 0xeb, 0x24, 0xcd, 0x36, 0x93, 0x36, 0x58, 0x96, 0x90, 0x72, 0x41, 0x0a, 0x8b, 0x84,
	0xa2, 0x4a, 0xf4, 0xd0, 0xde, 0xb8, 0x71, 0x42, 0x15, 0x54, 0x20, 0x93, 0x72, 0x8d, 0x42, 0x32,
	0x02, 0x2b, 0xbb, 0x76, 0x6a, 0x3b, 0x4b, 0xf2, 0x42, 0xbc, 0x0e, 0xaf, 0x84, 0xec, 0xa4, 0xcb,
	0x9f, 0x03, 0x37, 0xcf, 0x6f, 0x26, 0xf3, 0x7d, 0xf3, 0x29, 0x90, 0x76, 0x3b, 0xd5, 0xf6, 0x57,
	0x83, 0x56, 0x56, 0xb1, 0x53, 0x5f, 0x6c, 0x7f, 0x10, 0xd8, 0x70, 0x7c, 0x18, 0xd1, 0x58, 0x96,
	0xc3, 0xe6, 0x80, 0xda, 0x08, 0x25, 0x73, 0x52, 0x90, 0xf2, 0x82, 0x3f, 0x96, 0x2c, 0x83, 0x40,
	0x74, 0x79, 0x50, 0x90, 0x32, 0xe2, 0x81, 0xe8, 0xd8, 0x73, 0x38, 0x6f, 0xda, 0x16, 0x8d, 0xa9,
	0xad, 0xea, 0x51, 0xe6, 0x61, 0x41, 0xca, 0x84, 0xa7, 0x0b, 0xab, 0x1c, 0x62, 0x2f, 0x21, 0xb2,
	0xf3, 0x80, 0x79, 0x54, 0x90, 0x32, 0xbb, 0x66, 0x57, 0x8b, 0xf6, 0x2a, 0x55, 0xcd, 0x03, 0x72,
	0xdf, 0x77, 0x73, 0xae, 0x93, 0xdf, 0x14, 0xa4, 0x4c, 0xff, 0x9d, 0x7b, 0xaf, 0xda, 0x9e, 0xfb,
	0xfe, 0xf6, 0x27, 0x81, 0x33, 0x8e, 0x66, 0x50, 0xd2, 0xe0, 0x7f, 0x9c, 0x3e, 0x03, 0xd0, 0xcb,
	0xb7, 0xf5, 0xd1, 0x71, 0xb2, 0x92, 0xdb, 0x8e, 0xbd, 0x82, 0xd8, 0xd8, 0xc6, 0x8e, 0xc6, 0x5b,
	0xce, 0xae, 0x9f, 0x1e, 0xf5, 0x96, 0xcd, 0x9f, 0x7c, 0x93, 0xaf, 0x43, 0x6e, 0x1b, 0x6a, 0xad,
	0x74, 0x6d, 0x71, 0xb2, 0xfe, 0x94, 0x84, 0x27, 0x9e, 0x54, 0x38, 0x59, 0xc6, 0x20, 0xea, 0x71,
	0x36, 0xf9, 0x69, 0x11, 0x96, 0x09, 0xf7, 0x6f, 0x56, 0x02, 0x35, 0xa8, 0x0f, 0xa8, 0xeb, 0x51,
	0x8a, 0xa9, 0xb6, 0x62, 0x8f, 0x79, 0x5c, 0x90, 0x32, 0xe4, 0xd9, 0xc2, 0xef, 0xa5, 0x98, 0x2a,
	0xb1, 0xc7, 0x2d, 0x42, 0xfa, 0xc7, 0x99, 0x4e, 0xeb, 0x7b, 0x23, 0x6c, 0xbd, 0x17, 0xad, 0x56,
	0xfe, 0xac, 0x88, 0x27, 0x8e, 0xdc, 0x39, 0xc0, 0x5e, 0xc0, 0x85, 0xc6, 0x1d, 0x36, 0x06, 0xd7,
	0x89, 0xe5, 0xb6, 0xf3, 0x15, 0x2e, 0x43, 0x8f, 0x86, 0xc2, 0xdf, 0x86, 0x2e, 0x5f, 0x1f, 0x65,
	0x5c, 0xea, 0x2c, 0x85, 0xcd, 0xad, 0x3c, 0x34, 0x3b, 0xd1, 0xd1, 0x13, 0x76, 0x06, 0xd1, 0x47,
	0x21, 0xbf, 0x52, 0xe2, 0x5e, 0xce, 0x05, 0x0d, 0x18, 0x40, 0x7c, 0x2f, 0x5d, 0x28, 0x34, 0xbc,
	0xfc, 0x06, 0xd9, 0xdf, 0xc9, 0xb0, 0x18, 0x82, 0x0f, 0x3d, 0x3d, 0x71, 0x6b, 0xde, 0xa2, 0x44,
	0xdd, 0xec, 0x28, 0x71, 0xc5, 0xe7, 0x25, 0x7f, 0x1a, 0xb0, 0x27, 0x90, 0xae, 0x02, 0x4e, 0x8f,
	0x86, 0x0e, 0x54, 0x4a, 0xdd, 0x35, 0x72, 0x7e, 0x87, 0xb3, 0xa1, 0x1d, 0x63, 0x90, 0xbd, 0x69,
	0x1f, 0x46, 0xa1, 0xd1, 0xe5, 0xa0, 0x46, 0x4b, 0xa7, 0x2f, 0xb1, 0xff, 0x2b, 0x6f, 0x7e, 0x05,
	0x00, 0x00, 0xff, 0xff, 0x54, 0xd0, 0x7e, 0x1d, 0xa4, 0x02, 0x00, 0x00,
}
