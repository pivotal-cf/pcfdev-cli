// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin/cmd (interfaces: VBox)

package mocks

import (
	"github.com/golang/mock/gomock"
	"github.com/pivotal-cf/pcfdev-cli/config"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
)

// Mock of VBox interface
type MockVBox struct {
	ctrl     *gomock.Controller
	recorder *_MockVBoxRecorder
}

// Recorder for MockVBox (not exported)
type _MockVBoxRecorder struct {
	mock *MockVBox
}

func NewMockVBox(ctrl *gomock.Controller) *MockVBox {
	mock := &MockVBox{ctrl: ctrl}
	mock.recorder = &_MockVBoxRecorder{mock}
	return mock
}

func (_m *MockVBox) EXPECT() *_MockVBoxRecorder {
	return _m.recorder
}

func (_m *MockVBox) DestroyPCFDevVMs() error {
	ret := _m.ctrl.Call(_m, "DestroyPCFDevVMs")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockVBoxRecorder) DestroyPCFDevVMs() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "DestroyPCFDevVMs")
}

func (_m *MockVBox) GetVMName() (string, error) {
	ret := _m.ctrl.Call(_m, "GetVMName")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) GetVMName() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetVMName")
}

func (_m *MockVBox) VMConfig(_param0 string) (*config.VMConfig, error) {
	ret := _m.ctrl.Call(_m, "VMConfig", _param0)
	ret0, _ := ret[0].(*config.VMConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) VMConfig(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "VMConfig", arg0)
}

func (_m *MockVBox) Version() (*vboxdriver.VBoxDriverVersion, error) {
	ret := _m.ctrl.Call(_m, "Version")
	ret0, _ := ret[0].(*vboxdriver.VBoxDriverVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVBoxRecorder) Version() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Version")
}
