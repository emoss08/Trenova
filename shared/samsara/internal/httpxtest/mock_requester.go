package httpxtest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

var ErrDoFuncNotSet = errors.New("mock requester DoFunc is nil")

type MockRequester struct {
	DoFunc func(ctx context.Context, req httpx.Request) error
}

func (m *MockRequester) Do(ctx context.Context, req httpx.Request) error {
	if m.DoFunc == nil {
		return ErrDoFuncNotSet
	}

	return m.DoFunc(ctx, req)
}
