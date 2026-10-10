package quotatx_test

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type txKey struct{}

type fakeConnection struct {
	transactions int
}

func (f *fakeConnection) DB() *bun.DB { return nil }

func (f *fakeConnection) DBForContext(context.Context) bun.IDB { return nil }

func (f *fakeConnection) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	f.transactions++
	return fn(context.WithValue(ctx, txKey{}, true), bun.Tx{})
}

func (f *fakeConnection) HealthCheck(context.Context) error { return nil }

func (f *fakeConnection) IsHealthy(context.Context) bool { return true }

func (f *fakeConnection) Close() error { return nil }

func request(meter platformcatalog.MeterKey) platformplanports.QuotaRequest {
	return platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Meter:      meter,
		Quantity:   1,
	}
}

func TestRunWithoutEnforcementOpensNoTransaction(t *testing.T) {
	t.Parallel()

	conn := &fakeConnection{}
	called := false
	err := quotatx.Run(t.Context(), conn, quotaservice.NewUnlimited(), func(ctx context.Context) error {
		called = true
		assert.Nil(t, ctx.Value(txKey{}))
		return nil
	}, request(platformcatalog.MeterTractorsTotal))

	require.NoError(t, err)
	assert.True(t, called)
	assert.Zero(t, conn.transactions)
}

func TestRunEnforcesInsideTheTransactionBeforeWriting(t *testing.T) {
	t.Parallel()

	conn := &fakeConnection{}
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	req := request(platformcatalog.MeterTrailersTotal)
	guard.EXPECT().
		Enforce(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx.Value(txKey{}) == true
		}), &req).
		Return(nil).
		Once()

	called := false
	err := quotatx.Run(t.Context(), conn, guard, func(ctx context.Context) error {
		called = true
		assert.Equal(t, true, ctx.Value(txKey{}))
		return nil
	}, req)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, 1, conn.transactions)
}

func TestRunDoesNotWriteWhenTheQuotaRefuses(t *testing.T) {
	t.Parallel()

	conn := &fakeConnection{}
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	req := request(platformcatalog.MeterLocationsTotal)
	refusal := errortypes.NewQuotaExceededError(string(req.Meter), 25, 25, "free_demo")
	guard.EXPECT().Enforce(mock.Anything, &req).Return(refusal).Once()

	err := quotatx.Run(t.Context(), conn, guard, func(context.Context) error {
		t.Fatal("the write must not run once the quota refuses")
		return nil
	}, req)

	require.ErrorIs(t, err, refusal)
}

func TestRunReturnsTheWriteError(t *testing.T) {
	t.Parallel()

	conn := &fakeConnection{}
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	req := request(platformcatalog.MeterCustomersTotal)
	boom := errors.New("insert failed")
	guard.EXPECT().Enforce(mock.Anything, &req).Return(nil).Once()

	err := quotatx.Run(t.Context(), conn, guard, func(context.Context) error {
		return boom
	}, req)

	require.ErrorIs(t, err, boom)
}
