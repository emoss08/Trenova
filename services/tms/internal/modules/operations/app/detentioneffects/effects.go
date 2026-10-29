package detentioneffects

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Effects struct {
	l             *zap.Logger
	evidenceRepo  detentionports.DetentionEvidenceRepository
	organizations tenantports.OrganizationRepository
	auditService  auditports.AuditService
	realtime      realtimeports.RealtimeService
	now           func() int64
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	EvidenceRepo  detentionports.DetentionEvidenceRepository
	Organizations tenantports.OrganizationRepository
	AuditService  auditports.AuditService
	Realtime      realtimeports.RealtimeService `optional:"true"`
	Now           func() int64                  `optional:"true"`
}

func New(p Params) *Effects {
	now := p.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Effects{
		l:             p.Logger.Named("service.detention"),
		evidenceRepo:  p.EvidenceRepo,
		organizations: p.Organizations,
		auditService:  p.AuditService,
		realtime:      p.Realtime,
		now:           now,
	}
}

func (s *Effects) AppendEvidence(
	ctx context.Context,
	occurrence *detention.DetentionOccurrence,
	kind detentionevidence.EvidenceKind,
	source detentionevidence.EvidenceSource,
	summary string,
	observedAt int64,
) {
	if s.evidenceRepo == nil {
		return
	}

	entry := &detentionevidence.DetentionEvidence{
		OrganizationID:        occurrence.OrganizationID,
		BusinessUnitID:        occurrence.BusinessUnitID,
		DetentionOccurrenceID: occurrence.ID,
		Kind:                  kind,
		Source:                source,
		Summary:               summary,
		ObservedAt:            observedAt,
		RecordedAt:            s.now(),
	}

	if _, err := s.evidenceRepo.Append(ctx, &detentionports.AppendEvidenceRequest{
		Entry: entry,
	}); err != nil {
		s.l.Warn("failed to append detention evidence",
			zap.String("occurrenceId", occurrence.ID.String()), zap.Error(err))
	}
}

// TenantLocation resolves the organization's timezone so calendar-day caps
// break at the carrier's local midnight rather than UTC's.
func (s *Effects) TenantLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	return orgzone.Location(ctx, s.organizations, tenantInfo)
}

// PublishBillingHoldChange refreshes the billing queue when an occurrence
// starts or stops holding its shipment off an invoice, so a biller looking at
// the item sees the approve action unlock the moment the charge is decided. It
// waits for the write to commit when there is a transaction.
func (s *Effects) PublishBillingHoldChange(
	ctx context.Context,
	previous, saved *detention.DetentionOccurrence,
	actorUserID pulid.ID,
) {
	if s.realtime == nil || saved == nil {
		return
	}

	wasHeld := previous != nil && previous.HoldsBilling()
	if wasHeld == saved.HoldsBilling() {
		return
	}

	orgID, buID, shipmentID := saved.OrganizationID, saved.BusinessUnitID, saved.ShipmentID
	kernelports.AfterCommit(ctx, func(runCtx context.Context) {
		if err := realtimeinvalidation.Publish(
			runCtx,
			s.realtime,
			&realtimeinvalidation.PublishParams{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				ActorUserID:    actorUserID,
				Resource:       permission.ResourceBillingQueue.String(),
				Action:         "updated",
				Entity:         map[string]string{"shipmentId": shipmentID.String()},
			},
		); err != nil {
			s.l.Warn("failed to publish billing queue invalidation for a detention hold",
				zap.String("occurrenceId", saved.ID.String()), zap.Error(err))
		}
	})
}

func (s *Effects) Audit(
	original, updated *detention.DetentionOccurrence,
	userID pulid.ID,
	comment string,
) {
	if s.auditService == nil {
		return
	}

	if err := s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDetention,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment(comment),
		auditservice.WithDiff(original, updated),
	); err != nil {
		s.l.Error("failed to log detention audit action", zap.Error(err))
	}
}
