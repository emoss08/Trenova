package scopedvalidation

import (
	"context"

	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/stretchr/testify/mock"
)

type mockUniquenessChecker struct {
	mock.Mock
}

func newMockUniquenessChecker() *mockUniquenessChecker {
	return &mockUniquenessChecker{}
}

func (m *mockUniquenessChecker) CheckUniqueness(
	ctx context.Context,
	req *validationframework.UniquenessRequest,
) (bool, error) {
	args := m.Called(ctx, req)
	return args.Bool(0), args.Error(1)
}
