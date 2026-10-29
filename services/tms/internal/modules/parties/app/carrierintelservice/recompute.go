package carrierintelservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	recomputeBatchSize = 500
)

type RecomputeResult struct {
	Evaluated int `json:"evaluated"`
	Changed   int `json:"changed"`
}

func (s *Service) RecomputeStalePolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	deadline time.Time,
) (*RecomputeResult, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &RecomputeResult{}, nil
	}

	result := &RecomputeResult{}
	afterID := pulid.Nil
	for time.Now().Before(deadline) {
		page, listErr := s.snapshotRepo.ListForRecompute(
			ctx,
			&carrierintelports.ListSnapshotsForRecomputeRequest{
				TenantInfo:   tenantInfo,
				BelowVersion: control.PolicyVersion,
				AfterID:      afterID,
				Limit:        recomputeBatchSize,
			},
		)
		if listErr != nil {
			return result, listErr
		}
		if len(page) == 0 {
			return result, nil
		}

		for _, snap := range page {
			afterID = snap.ID
			subject, subjectErr := s.subjectForSnapshot(ctx, tenantInfo, snap)
			if subjectErr != nil {
				s.l.Warn("failed to resolve subject for recompute", zap.Error(subjectErr))
				continue
			}
			ingested, ingestErr := s.confirmUnchanged(ctx, &ingestInput{
				tenant:  tenantInfo,
				control: control,
				subject: subject,
				bound:   &BoundProvider{provider: provider, tenant: tenantInfo},
				result:  &carrierintelports.CarrierIntelLookupResult{},
				current: snap,
				source:  carrierintelevent.EventSourceRuleEvaluation,
				purpose: carrierintel.PurposeReconcile,
			}, snap, s.recomputeFindings(ctx, tenantInfo, control, subject, snap), snap.EffectiveAsOf())
			if ingestErr != nil {
				s.l.Warn("failed to recompute snapshot findings", zap.Error(ingestErr))
				continue
			}
			result.Evaluated++
			if len(ingested.Raised) > 0 {
				result.Changed++
			}
		}
		if len(page) < recomputeBatchSize {
			return result, nil
		}
	}
	return result, nil
}

func (s *Service) recomputeFindings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintelcontrol.CarrierIntelControl,
	subject carrierintelports.CarrierIntelSubject,
	snap *carrierintelsnapshot.CarrierIntelSnapshot,
) []carrierintelrule.Finding {
	findings, err := s.evaluateSubject(
		ctx,
		tenantInfo,
		control,
		subject,
		snap.Profile,
		snap.NotFound,
	)
	if err != nil {
		return snap.Findings
	}
	return carrierintelrule.CarryUnconfirmed(snap.Findings, findings)
}

func (s *Service) subjectForSnapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snap *carrierintelsnapshot.CarrierIntelSnapshot,
) (carrierintelports.CarrierIntelSubject, error) {
	switch snap.SubjectType {
	case carrierintel.SubjectTypeCarrier:
		subjects, err := s.subjectRepo.ListCarrierSubjects(
			ctx,
			&carrierintelports.ListCarrierIntelSubjectsRequest{
				TenantInfo: tenantInfo,
				CarrierIDs: []pulid.ID{snap.CarrierID},
				Limit:      1,
			},
		)
		if err != nil {
			return carrierintelports.CarrierIntelSubject{}, err
		}
		if len(subjects) == 1 {
			return subjects[0], nil
		}
	case carrierintel.SubjectTypeOrganization:
		org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
		if err != nil {
			return carrierintelports.CarrierIntelSubject{}, err
		}
		if org != nil {
			return *org, nil
		}
	case carrierintel.SubjectTypeCustomer:
		customerID, err := pulid.Parse(snap.SubjectID)
		if err == nil {
			customer, customerErr := s.subjectRepo.GetCustomerSubject(ctx, tenantInfo, customerID)
			if customerErr == nil {
				return *customer, nil
			}
		}
	}
	return carrierintelports.CarrierIntelSubject{
		SubjectType:  snap.SubjectType,
		SubjectID:    snap.SubjectID,
		CarrierID:    snap.CarrierID,
		DOTNumber:    snap.DOTNumber,
		DocketNumber: snap.DocketNumber,
	}, nil
}

func (s *Service) RefreshStoredEvaluation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) {
	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		tenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   carrierID.String(),
		},
	)
	if err != nil || snapshot == nil {
		return
	}
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return
	}
	subject, err := s.subjectForSnapshot(ctx, tenantInfo, snapshot)
	if err != nil {
		return
	}
	findings := s.recomputeFindings(ctx, tenantInfo, control, subject, snapshot)
	prior := snapshot.Findings
	snapshot.ApplyFindings(findings, control.PolicyVersion)
	snapshot.ReviewState = nextReviewState(snapshot, prior, findings)
	if err = s.snapshotRepo.UpdateEvaluation(ctx, snapshot); err != nil {
		s.l.Warn("failed to refresh stored findings after override change", zap.Error(err))
		return
	}
	if err = s.UpdateCarrierSummary(ctx, tenantInfo, subject, snapshot); err != nil {
		s.l.Warn("failed to update carrier summary after override change", zap.Error(err))
	}
	s.Publish(ctx, tenantInfo, "carriers", "updated", carrierID)
	s.Publish(ctx, tenantInfo, "carrier_intelligence", "updated", snapshot.ID)
}
