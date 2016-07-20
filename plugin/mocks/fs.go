// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin (interfaces: FS)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
)

// Mock of FS interface
type MockFS struct {
	ctrl     *gomock.Controller
	recorder *_MockFSRecorder
}

// Recorder for MockFS (not exported)
type _MockFSRecorder struct {
	mock *MockFS
}

func NewMockFS(ctrl *gomock.Controller) *MockFS {
	mock := &MockFS{ctrl: ctrl}
	mock.recorder = &_MockFSRecorder{mock}
	return mock
}

func (_m *MockFS) EXPECT() *_MockFSRecorder {
	return _m.recorder
}

func (_m *MockFS) Copy(_param0 string, _param1 string) error {
	ret := _m.ctrl.Call(_m, "Copy", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) Copy(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Copy", arg0, arg1)
}

func (_m *MockFS) MD5(_param0 string) (string, error) {
	ret := _m.ctrl.Call(_m, "MD5", _param0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockFSRecorder) MD5(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "MD5", arg0)
}

func (_m *MockFS) Remove(_param0 string) error {
	ret := _m.ctrl.Call(_m, "Remove", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) Remove(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Remove", arg0)
}
