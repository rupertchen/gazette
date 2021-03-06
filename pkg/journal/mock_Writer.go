// Code generated by mockery v1.0.0
package journal

import io "io"
import mock "github.com/stretchr/testify/mock"

// MockWriter is an autogenerated mock type for the Writer type
type MockWriter struct {
	mock.Mock
}

// ReadFrom provides a mock function with given fields: journal, r
func (_m *MockWriter) ReadFrom(journal Name, r io.Reader) (*AsyncAppend, error) {
	ret := _m.Called(journal, r)

	var r0 *AsyncAppend
	if rf, ok := ret.Get(0).(func(Name, io.Reader) *AsyncAppend); ok {
		r0 = rf(journal, r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*AsyncAppend)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(Name, io.Reader) error); ok {
		r1 = rf(journal, r)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Write provides a mock function with given fields: journal, buffer
func (_m *MockWriter) Write(journal Name, buffer []byte) (*AsyncAppend, error) {
	ret := _m.Called(journal, buffer)

	var r0 *AsyncAppend
	if rf, ok := ret.Get(0).(func(Name, []byte) *AsyncAppend); ok {
		r0 = rf(journal, buffer)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*AsyncAppend)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(Name, []byte) error); ok {
		r1 = rf(journal, buffer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
