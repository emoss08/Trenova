package briefingjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// retentionDays is how far back a reader can page. A fortnight covers
	// coming back from leave; past that the records themselves are the
	// record.
	retentionDays  = 60
	retentionBatch = 1000
	// retentionPasses bounds one run's deletes so a first sweep over a
	// long-running deployment ends in a predictable time.
	retentionPasses = 10
)

type ActivitiesParams struct {
	fx.In

	Logger         *zap.Logger
	Briefings      briefingports.BriefingService
	Repo           briefingports.BriefingRepository
	Board          shipmentbriefports.ShipmentBriefWriter
	BoardRepo      shipmentbriefports.ShipmentBriefRepository
	Accounting     accountingbriefports.AccountingBriefWriter
	AccountingRepo accountingbriefports.AccountingBriefRepository
	Controls       tenantports.AgentControlRepository
	Tenants        tenantports.TenantSyncRepository
	Orgs           tenantports.OrganizationRepository
	Plans          platformplanports.PlanService `optional:"true"`
}

type Activities struct {
	l              *zap.Logger
	briefings      briefingports.BriefingService
	repo           briefingports.BriefingRepository
	board          shipmentbriefports.ShipmentBriefWriter
	boardRepo      shipmentbriefports.ShipmentBriefRepository
	accounting     accountingbriefports.AccountingBriefWriter
	accountingRepo accountingbriefports.AccountingBriefRepository
	controls       tenantports.AgentControlRepository
	tenants        tenantports.TenantSyncRepository
	orgs           tenantports.OrganizationRepository
	plans          platformplanports.PlanService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		l:              p.Logger.Named("job.daily-briefing"),
		briefings:      p.Briefings,
		repo:           p.Repo,
		board:          p.Board,
		boardRepo:      p.BoardRepo,
		accounting:     p.Accounting,
		accountingRepo: p.AccountingRepo,
		controls:       p.Controls,
		tenants:        p.Tenants,
		orgs:           p.Orgs,
		plans:          p.Plans,
	}
}

// ListDueOrganizationsActivity decides which organizations' briefing hour has
// come as of the sweep's instant. One whose hour cannot be decided is named,
// and the rest are still listed.
func (a *Activities) ListDueOrganizationsActivity(
	ctx context.Context,
	input *DueOrganizationsInput,
) (*DueOrganizations, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	result := &DueOrganizations{Due: make([]temporaljobs.TenantWorkItem, 0)}
	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		due, dErr := a.isDue(tenantCtx, tenantInfo, input.Now)
		if dErr != nil {
			a.l.Warn("could not decide whether a briefing is due",
				zap.String("organization", org.ID.String()),
				zap.Error(dErr),
			)
			result.Failed = append(result.Failed, org.ID.String())

			continue
		}
		if due {
			result.Due = append(result.Due, temporaljobs.NewTenantWorkItem(tenantInfo, 1))
		}
	}

	return result, nil
}

// WriteOrganizationBriefingActivity writes one organization's morning as of
// the sweep's instant. It heartbeats on a timer, because a model call per
// role can outlast the heartbeat timeout without producing anything.
func (a *Activities) WriteOrganizationBriefingActivity(
	ctx context.Context,
	input *OrganizationBriefingInput,
) (*OrganizationBriefingResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	written, err := a.briefings.WriteForDay(ctx, briefingports.WriteBriefingRequest{
		TenantInfo: input.TenantInfo(),
		Now:        input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("write the morning briefing: %w", err)
	}

	return &OrganizationBriefingResult{Written: written.Written, Narrated: written.Narrated}, nil
}

func (a *Activities) WriteShipmentBriefActivity(
	ctx context.Context,
	input *OrganizationBriefingInput,
) (*ShipmentBriefResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	brief, err := a.board.WriteBrief(ctx, &shipmentbriefports.WriteShipmentBriefRequest{
		TenantInfo: input.TenantInfo(),
		Trigger:    shipmentbrief.TriggerScheduled,
	})
	if err != nil {
		return nil, fmt.Errorf("write the shipment board brief: %w", err)
	}

	return &ShipmentBriefResult{Generation: brief.Generation, Narrated: brief.Narrated}, nil
}

func (a *Activities) WriteAccountingBriefActivity(
	ctx context.Context,
	input *OrganizationBriefingInput,
) (*AccountingBriefResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	brief, err := a.accounting.WriteBrief(ctx, &accountingbriefports.WriteAccountingBriefRequest{
		TenantInfo: input.TenantInfo(),
		Trigger:    accountingbrief.TriggerScheduled,
	})
	if err != nil {
		return nil, fmt.Errorf("write the accounting brief: %w", err)
	}

	return &AccountingBriefResult{Generation: brief.Generation, Narrated: brief.Narrated}, nil
}

// BriefingRetentionActivity removes briefings older than the window a reader can
// page back through.
func (a *Activities) BriefingRetentionActivity(
	ctx context.Context,
) (*BriefingRetentionResult, error) {
	cutoff := time.Unix(timeutils.NowUnix(), 0).
		UTC().
		AddDate(0, 0, -retentionDays).
		Format(timeutils.ISODateLayout)
	result := &BriefingRetentionResult{}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.repo.DeleteBefore(ctx, briefingports.DeleteBriefingsBeforeRequest{
			BeforeDate: cutoff,
			Limit:      retentionBatch,
		})
		if err != nil {
			return nil, fmt.Errorf("delete briefings: %w", err)
		}
		result.Deleted += deleted
		if deleted < retentionBatch {
			break
		}
	}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.boardRepo.DeleteBefore(
			ctx,
			&shipmentbriefports.DeleteShipmentBriefsBeforeRequest{
				BeforeDate: cutoff,
				Limit:      retentionBatch,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("delete shipment board briefs: %w", err)
		}
		result.DeletedBoardBriefs += deleted
		if deleted < retentionBatch {
			break
		}
	}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.accountingRepo.DeleteBefore(
			ctx,
			&accountingbriefports.DeleteAccountingBriefsBeforeRequest{
				BeforeDate: cutoff,
				Limit:      retentionBatch,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("delete accounting briefs: %w", err)
		}
		result.DeletedAccountingBriefs += deleted
		if deleted < retentionBatch {
			break
		}
	}

	a.l.Info("briefing retention sweep complete",
		zap.Int("deleted", result.Deleted),
		zap.Int("deletedBoardBriefs", result.DeletedBoardBriefs),
		zap.Int("deletedAccountingBriefs", result.DeletedAccountingBriefs),
	)

	return result, nil
}

// isDue reports whether this organization's local clock has just reached
// its briefing hour. The comparison is on the hour rather than the minute
// because the sweep runs hourly: a tenant is written in the firing whose
// local hour matches, and in no other.
func (a *Activities) isDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (bool, error) {
	allowed, err := planservice.Allows(
		ctx,
		a.plans,
		tenantInfo,
		platformplan.CapabilityAgentAutomation,
	)
	if err != nil || !allowed {
		return false, err
	}

	control, err := a.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	if !control.BriefingEnabled {
		return false, nil
	}

	location, err := orgzone.Location(ctx, a.orgs, tenantInfo)
	if err != nil {
		return false, err
	}

	return time.Unix(now, 0).In(location).Hour() == briefingHour(control), nil
}

func briefingHour(control *tenant.AgentControl) int {
	if control.BriefingHourLocal < 0 || control.BriefingHourLocal > 23 {
		return tenant.DefaultBriefingHourLocal
	}

	return control.BriefingHourLocal
}
