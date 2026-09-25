package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type previewDefinitions struct {
	repositories.ReportDefinitionRepository

	definition *report.ReportDefinition
}

func (r *previewDefinitions) GetByID(
	context.Context,
	*repositories.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	return r.definition, nil
}

type previewSchedules struct {
	repositories.ReportScheduleRepository

	created  *report.ReportSchedule
	writable bool
}

func (r *previewSchedules) Create(
	_ context.Context,
	entity *report.ReportSchedule,
) (*report.ReportSchedule, error) {
	if !r.writable {
		return nil, errors.New("a preview wrote")
	}
	r.created = entity

	return entity, nil
}

type previewOrganizations struct {
	repositories.OrganizationRepository
}

func (previewOrganizations) GetByID(
	context.Context,
	repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

func TestPreviewSchedule_IsTheScheduleCreateScheduleSaves(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	definition := &report.ReportDefinition{
		ID:         pulid.MustNew("rdef_"),
		Name:       "Unbilled aging",
		OwnerID:    userID,
		Visibility: report.VisibilityShared,
	}
	schedules := &previewSchedules{}
	svc := &Service{
		l:            zap.NewNop(),
		defRepo:      &previewDefinitions{definition: definition},
		scheduleRepo: schedules,
		orgRepo:      previewOrganizations{},
	}
	request := func() *SaveScheduleRequest {
		return &SaveScheduleRequest{
			Request: Request{TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: userID,
			}},
			DefinitionID:    definition.ID,
			CronExpression:  "0 7 * * 1",
			Formats:         []string{"csv"},
			EmailRecipients: []string{"ops@carrier.example", "ops@carrier.example"},
			EmailAttach:     true,
			Enabled:         true,
		}
	}

	req := request()
	preview, err := svc.PreviewSchedule(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, req.Timezone, "a preview must not rewrite the caller's request")

	schedules.writable = true
	created, err := svc.CreateSchedule(t.Context(), request())
	require.NoError(t, err)

	assert.Equal(t, "Unbilled aging", preview.Definition.Name)
	assert.Equal(t, created.CronExpression, preview.Schedule.CronExpression)
	assert.Equal(t, "America/Chicago", preview.Schedule.Timezone)
	assert.Equal(t, created.Timezone, preview.Schedule.Timezone)
	assert.Equal(t, created.Delivery, preview.Schedule.Delivery)
	assert.Equal(t, []string{"ops@carrier.example"}, preview.Schedule.Delivery.EmailRecipients)
	assert.Equal(t, created.NextRunAt, preview.Schedule.NextRunAt)
}
