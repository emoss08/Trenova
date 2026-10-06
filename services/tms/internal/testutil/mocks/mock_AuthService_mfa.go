package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	mock "github.com/stretchr/testify/mock"
)

func (_mock *MockAuthService) VerifyMFAChallenge(
	ctx context.Context,
	req services.VerifyMFAChallengeRequest,
) (*services.LoginResponse, error) {
	ret := _mock.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for VerifyMFAChallenge")
	}

	if returnFunc, ok := ret.Get(0).(func(context.Context, services.VerifyMFAChallengeRequest) (*services.LoginResponse, error)); ok {
		return returnFunc(ctx, req)
	}

	var r0 *services.LoginResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*services.LoginResponse)
	}

	return r0, ret.Error(1)
}

type MockAuthService_VerifyMFAChallenge_Call struct {
	*mock.Call
}

func (_e *MockAuthService_Expecter) VerifyMFAChallenge(
	ctx any,
	req any,
) *MockAuthService_VerifyMFAChallenge_Call {
	return &MockAuthService_VerifyMFAChallenge_Call{
		Call: _e.mock.On("VerifyMFAChallenge", ctx, req),
	}
}

func (_c *MockAuthService_VerifyMFAChallenge_Call) Return(
	loginResponse *services.LoginResponse,
	err error,
) *MockAuthService_VerifyMFAChallenge_Call {
	_c.Call.Return(loginResponse, err)
	return _c
}
