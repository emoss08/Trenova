package tenantportsmock

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	mock "github.com/stretchr/testify/mock"
)

func (_mock *MockAuthService) VerifyMFAChallenge(
	ctx context.Context,
	req tenantports.VerifyMFAChallengeRequest,
) (*tenantports.LoginResponse, error) {
	ret := _mock.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for VerifyMFAChallenge")
	}

	if returnFunc, ok := ret.Get(0).(func(context.Context, tenantports.VerifyMFAChallengeRequest) (*tenantports.LoginResponse, error)); ok {
		return returnFunc(ctx, req)
	}

	var r0 *tenantports.LoginResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*tenantports.LoginResponse)
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
	loginResponse *tenantports.LoginResponse,
	err error,
) *MockAuthService_VerifyMFAChallenge_Call {
	_c.Call.Return(loginResponse, err)
	return _c
}
