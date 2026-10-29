package reportscheduleservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type previewDefinitions struct {
	reportports.ReportDefinitionRepository

	definition *report.ReportDefinition
}

func (r *previewDefinitions) GetByID(
	context.Context,
	*reportports.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	return r.definition, nil
}

type previewSchedules struct {
	reportports.ReportScheduleRepository

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

func scheduleCore(
	t *testing.T,
	definitions reportports.ReportDefinitionRepository,
) *reporting.Service {
	t.Helper()
	core, err := reporting.New(reporting.Params{
		Logger:           zap.NewNop(),
		DefinitionRepo:   definitions,
		OrganizationRepo: reportingtest.Organizations{},
		Engine:           reportingtest.NewEngine(reportingtest.ReportCatalog(t), nil),
		ReportingConfig:  &reportingconfig.Config{},
	})
	require.NoError(t, err)
	return core
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
		reporting:    scheduleCore(t, &previewDefinitions{definition: definition}),
		scheduleRepo: schedules,
	}
	request := func() *SaveScheduleRequest {
		return &SaveScheduleRequest{
			Request: reporting.Request{TenantInfo: pagination.TenantInfo{
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
