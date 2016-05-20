// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin (interfaces: Client)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
)

// Mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *_MockClientRecorder
}

// Recorder for MockClient (not exported)
type _MockClientRecorder struct {
	mock *MockClient
}

func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &_MockClientRecorder{mock}
	return mock
}

func (_m *MockClient) EXPECT() *_MockClientRecorder {
	return _m.recorder
}

func (_m *MockClient) AcceptEULA() error {
	ret := _m.ctrl.Call(_m, "AcceptEULA")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockClientRecorder) AcceptEULA() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "AcceptEULA")
}

func (_m *MockClient) GetEULA() (string, error) {
	ret := _m.ctrl.Call(_m, "GetEULA")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockClientRecorder) GetEULA() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetEULA")
}

func (_m *MockClient) IsEULAAccepted() (bool, error) {
	ret := _m.ctrl.Call(_m, "IsEULAAccepted")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockClientRecorder) IsEULAAccepted() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "IsEULAAccepted")
}
