package report

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ReportSchedule)(nil)
	_ validationframework.TenantedEntity = (*ReportSchedule)(nil)
)

type ScheduleDelivery struct {
	EmailRecipients []string `json:"emailRecipients,omitempty"`
}

type ReportSchedule struct {
	bun.BaseModel `bun:"table:report_schedules,alias:rsch" json:"-"`

	ID                  pulid.ID          `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID      pulid.ID          `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID          `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DefinitionID        pulid.ID          `json:"definitionId"        bun:"definition_id,type:VARCHAR(100),notnull"`
	CronExpression      string            `json:"cronExpression"      bun:"cron_expression,type:VARCHAR(100),notnull"`
	Timezone            string            `json:"timezone"            bun:"timezone,type:VARCHAR(64),notnull"`
	Formats             []string          `json:"formats"             bun:"formats,type:TEXT[],array,notnull"`
	Delivery            *ScheduleDelivery `json:"delivery"            bun:"delivery,type:JSONB,nullzero"`
	Enabled             bool              `json:"enabled"             bun:"enabled,type:BOOLEAN,notnull,default:true"`
	RunAsID             pulid.ID          `json:"runAsId"             bun:"run_as_id,type:VARCHAR(100),notnull"`
	LastRunID           pulid.ID          `json:"lastRunId"           bun:"last_run_id,type:VARCHAR(100),nullzero"`
	NextRunAt           int64             `json:"nextRunAt"           bun:"next_run_at,type:BIGINT,nullzero"`
	ConsecutiveFailures int               `json:"consecutiveFailures" bun:"consecutive_failures,type:INTEGER,notnull,default:0"`
	Version             int64             `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64             `json:"createdAt"           bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64             `json:"updatedAt"           bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	RunAs            *tenant.User         `json:"runAs,omitempty"            bun:"rel:belongs-to,join:run_as_id=id"`
	ReportDefinition *ReportDefinition    `json:"reportDefinition,omitempty" bun:"rel:belongs-to,join:definition_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (rs *ReportSchedule) Validate(multiErr *errortypes.MultiError) {
	if rs.CronExpression == "" {
		multiErr.Add("cronExpression", errortypes.ErrRequired, "Cron expression is required")
	}
	if rs.Timezone == "" {
		multiErr.Add("timezone", errortypes.ErrRequired, "Timezone is required")
	}
	if len(rs.Formats) == 0 {
		multiErr.Add("formats", errortypes.ErrRequired, "At least one format is required")
	}
	for i, format := range rs.Formats {
		if !Format(format).IsValid() {
			multiErr.Add(fmt.Sprintf("formats[%d]", i), errortypes.ErrInvalid, "Format is invalid")
		}
	}
	if rs.RunAsID.IsNil() {
		multiErr.Add("runAsId", errortypes.ErrRequired, "Run-as user is required")
	}
}

func (rs *ReportSchedule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rs.ID.IsNil() {
			rs.ID = pulid.MustNew("rsch_")
		}
		rs.CreatedAt = now
	case *bun.UpdateQuery:
		rs.UpdatedAt = now
	}

	return nil
}

func (rs *ReportSchedule) GetID() pulid.ID { return rs.ID }

func (rs *ReportSchedule) GetOrganizationID() pulid.ID { return rs.OrganizationID }

func (rs *ReportSchedule) GetBusinessUnitID() pulid.ID { return rs.BusinessUnitID }

func (rs *ReportSchedule) GetTableName() string { return "report_schedules" }
