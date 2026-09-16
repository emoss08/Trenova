package fiscalyearservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func openYear() *fiscalyear.FiscalYear {
	closedAt := int64(1_798_761_600)

	return &fiscalyear.FiscalYear{
		ID:                    pulid.MustNew("fy_"),
		OrganizationID:        pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		Status:                fiscalyear.StatusOpen,
		Year:                  2026,
		Name:                  "FY 2026",
		Description:           "Primary books",
		StartDate:             1_767_225_600,
		EndDate:               1_798_761_599,
		IsCurrent:             true,
		IsCalendarYear:        true,
		AllowAdjustingEntries: true,
		ClosedAt:              &closedAt,
		ClosedByID:            pulid.MustNew("usr_"),
	}
}

func fieldMessages(t *testing.T, multiErr *errortypes.MultiError) map[string]string {
	t.Helper()

	require.NotNil(t, multiErr)
	messages := make(map[string]string, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		messages[e.Field] = e.Error()
	}

	return messages
}

func TestValidateEditableAllowsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	original := openYear()
	updated := *original
	updated.Description = ""
	updated.AllowAdjustingEntries = false
	updated.Name = "FY 2026 Books"

	assert.Nil(t, validateEditable(original, &updated))
}

func TestValidateEditableRejectsCalendarChanges(t *testing.T) {
	t.Parallel()

	original := openYear()
	original.Status = fiscalyear.StatusDraft
	updated := *original
	updated.Year = 2027
	updated.StartDate += 86_400
	updated.EndDate += 86_400
	updated.IsCalendarYear = false

	messages := fieldMessages(t, validateEditable(original, &updated))

	for _, field := range []string{"year", "startDate", "endDate", "isCalendarYear"} {
		assert.Contains(t, messages[field], "cannot be changed after the fiscal year is created", field)
	}
}

func TestValidateEditableRejectsRenamingClosedYear(t *testing.T) {
	t.Parallel()

	original := openYear()
	original.Status = fiscalyear.StatusClosed
	updated := *original
	updated.Name = "Renamed"

	messages := fieldMessages(t, validateEditable(original, &updated))

	assert.Contains(t, messages["name"], "closed fiscal year cannot be changed")
}

func TestValidateEditableRejectsPermanentlyClosedYear(t *testing.T) {
	t.Parallel()

	original := openYear()
	original.Status = fiscalyear.StatusPermanentlyClosed
	updated := *original
	updated.Description = "Late note"

	messages := fieldMessages(t, validateEditable(original, &updated))

	assert.Contains(t, messages["status"], "Permanently closed fiscal years cannot be changed")
}

func TestPreserveLifecycleIgnoresWorkflowFieldsFromThePayload(t *testing.T) {
	t.Parallel()

	original := openYear()
	updated := *original
	updated.Status = fiscalyear.StatusPermanentlyClosed
	updated.IsCurrent = false
	updated.ClosedAt = nil
	updated.ClosedByID = pulid.Nil
	updated.ReopenReason = "forged"
	updated.Periods = []*fiscalperiod.FiscalPeriod{{Status: fiscalperiod.StatusClosed}}

	preserveLifecycle(original, &updated)

	assert.Equal(t, fiscalyear.StatusOpen, updated.Status)
	assert.True(t, updated.IsCurrent)
	assert.Equal(t, original.ClosedAt, updated.ClosedAt)
	assert.Equal(t, original.ClosedByID, updated.ClosedByID)
	assert.Empty(t, updated.ReopenReason)
	assert.Nil(t, updated.Periods)
}

func TestValidateCloseCountsLockedPeriodsAsUnclosed(t *testing.T) {
	t.Parallel()

	fpRepo := mocks.NewMockFiscalPeriodRepository(t)
	service := &Service{l: zap.NewNop(), fiscalPeriodRepo: fpRepo}
	year := openYear()

	fpRepo.EXPECT().
		CountUnclosedPeriodsByFiscalYear(t.Context(), repositories.CountUnclosedPeriodsByFiscalYearRequest{
			FiscalYearID: year.ID,
			OrgID:        year.OrganizationID,
			BuID:         year.BusinessUnitID,
		}).
		Return(1, nil).
		Once()

	multiErr := service.validateClose(t.Context(), year)

	messages := fieldMessages(t, multiErr)
	assert.True(t, strings.Contains(messages["status"], "1 period(s) are still open or locked"))
}

func TestNormalizeDateBoundsSnapsToWholeUTCDays(t *testing.T) {
	t.Parallel()

	entity := &fiscalyear.FiscalYear{
		StartDate: 1_767_243_600,
		EndDate:   1_798_675_200,
	}

	require.NoError(t, normalizeDateBounds(entity))

	assert.Equal(t, int64(1_767_225_600), entity.StartDate)
	assert.Equal(t, int64(1_798_761_599), entity.EndDate)
}

func TestNormalizeDateBoundsLeavesMissingDatesForValidation(t *testing.T) {
	t.Parallel()

	entity := &fiscalyear.FiscalYear{}

	require.NoError(t, normalizeDateBounds(entity))

	assert.Zero(t, entity.StartDate)
	assert.Zero(t, entity.EndDate)
}
