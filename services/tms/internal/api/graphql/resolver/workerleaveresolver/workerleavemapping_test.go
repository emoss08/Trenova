package workerleaveresolver

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int { return &v }

func TestLeaveControlRequest_CarriesTheVersionRead(t *testing.T) {
	t.Parallel()

	req, err := leaveControlRequest(&gqlmodel.UpdateLeaveControlInput{
		Version:              7,
		EligibilityMonths:    intPtr(12),
		EligibilityHours:     intPtr(1250),
		CertificationDueDays: intPtr(15),
	}, pagination.TenantInfo{}, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.Equal(t, int64(7), req.Version)
	require.NotNil(t, req.EligibilityMonths)
	assert.Equal(t, int32(12), *req.EligibilityMonths)
	require.NotNil(t, req.EligibilityHours)
	assert.Equal(t, int32(1250), *req.EligibilityHours)
	require.NotNil(t, req.CertificationDueDays)
	assert.Equal(t, int32(15), *req.CertificationDueDays)
}

func TestLeaveControlRequest_RefusesANegativeVersion(t *testing.T) {
	t.Parallel()

	_, err := leaveControlRequest(
		&gqlmodel.UpdateLeaveControlInput{Version: -1},
		pagination.TenantInfo{},
		pulid.MustNew("usr_"),
	)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "version", valErr.Path.String())
	assert.Equal(t, errortypes.ErrInvalid, valErr.Code)
}

func TestLeaveControlRequest_RefusesANumberThatWouldWrap(t *testing.T) {
	t.Parallel()

	_, err := leaveControlRequest(&gqlmodel.UpdateLeaveControlInput{
		CertificationDueDays: intPtr(math.MaxInt32 + 16),
	}, pagination.TenantInfo{}, pulid.MustNew("usr_"))

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "certificationDueDays", valErr.Path.String())
	assert.Equal(t, errortypes.ErrInvalid, valErr.Code)
	assert.Equal(t, "Must be between {0} and {1}", valErr.Message)
	assert.Equal(t, []any{int64(math.MinInt32), int64(math.MaxInt32)}, valErr.Args)
}
