// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin/cmd (interfaces: EULAUI)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
)

// Mock of EULAUI interface
type MockEULAUI struct {
	ctrl     *gomock.Controller
	recorder *_MockEULAUIRecorder
}

// Recorder for MockEULAUI (not exported)
type _MockEULAUIRecorder struct {
	mock *MockEULAUI
}

func NewMockEULAUI(ctrl *gomock.Controller) *MockEULAUI {
	mock := &MockEULAUI{ctrl: ctrl}
	mock.recorder = &_MockEULAUIRecorder{mock}
	return mock
}

func (_m *MockEULAUI) EXPECT() *_MockEULAUIRecorder {
	return _m.recorder
}

func (_m *MockEULAUI) Close() error {
	ret := _m.ctrl.Call(_m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockEULAUIRecorder) Close() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Close")
}

func (_m *MockEULAUI) ConfirmText(_param0 string) bool {
	ret := _m.ctrl.Call(_m, "ConfirmText", _param0)
	ret0, _ := ret[0].(bool)
	return ret0
}

func (_mr *_MockEULAUIRecorder) ConfirmText(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ConfirmText", arg0)
}

func (_m *MockEULAUI) Init() error {
	ret := _m.ctrl.Call(_m, "Init")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockEULAUIRecorder) Init() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Init")
}
