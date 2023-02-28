// Code generated by mockery v2.18.0. DO NOT EDIT.

package mocks

import (
	context "context"

	data "github.com/newscred/webhook-broker/storage/data"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MessageRepository is an autogenerated mock type for the MessageRepository type
type MessageRepository struct {
	mock.Mock
}

// Create provides a mock function with given fields: message
func (_m *MessageRepository) Create(message *data.Message) error {
	ret := _m.Called(message)

	var r0 error
	if rf, ok := ret.Get(0).(func(*data.Message) error); ok {
		r0 = rf(message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: channelID, messageID
func (_m *MessageRepository) Get(channelID string, messageID string) (*data.Message, error) {
	ret := _m.Called(channelID, messageID)

	var r0 *data.Message
	if rf, ok := ret.Get(0).(func(string, string) *data.Message); ok {
		r0 = rf(channelID, messageID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*data.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(channelID, messageID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID provides a mock function with given fields: id
func (_m *MessageRepository) GetByID(id string) (*data.Message, error) {
	ret := _m.Called(id)

	var r0 *data.Message
	if rf, ok := ret.Get(0).(func(string) *data.Message); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*data.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByIDs provides a mock function with given fields: ids
func (_m *MessageRepository) GetByIDs(ids []string) ([]*data.Message, error) {
	ret := _m.Called(ids)

	var r0 []*data.Message
	if rf, ok := ret.Get(0).(func([]string) []*data.Message); ok {
		r0 = rf(ids)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*data.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]string) error); ok {
		r1 = rf(ids)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMessagesForChannel provides a mock function with given fields: channelID, page
func (_m *MessageRepository) GetMessagesForChannel(channelID string, page *data.Pagination) ([]*data.Message, *data.Pagination, error) {
	ret := _m.Called(channelID, page)

	var r0 []*data.Message
	if rf, ok := ret.Get(0).(func(string, *data.Pagination) []*data.Message); ok {
		r0 = rf(channelID, page)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*data.Message)
		}
	}

	var r1 *data.Pagination
	if rf, ok := ret.Get(1).(func(string, *data.Pagination) *data.Pagination); ok {
		r1 = rf(channelID, page)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*data.Pagination)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, *data.Pagination) error); ok {
		r2 = rf(channelID, page)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetMessagesNotDispatchedForCertainPeriod provides a mock function with given fields: delta
func (_m *MessageRepository) GetMessagesNotDispatchedForCertainPeriod(delta time.Duration) []*data.Message {
	ret := _m.Called(delta)

	var r0 []*data.Message
	if rf, ok := ret.Get(0).(func(time.Duration) []*data.Message); ok {
		r0 = rf(delta)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*data.Message)
		}
	}

	return r0
}

// SetDispatched provides a mock function with given fields: txContext, message
func (_m *MessageRepository) SetDispatched(txContext context.Context, message *data.Message) error {
	ret := _m.Called(txContext, message)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *data.Message) error); ok {
		r0 = rf(txContext, message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMessageRepository interface {
	mock.TestingT
	Cleanup(func())
}

// NewMessageRepository creates a new instance of MessageRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMessageRepository(t mockConstructorTestingTNewMessageRepository) *MessageRepository {
	mock := &MessageRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
