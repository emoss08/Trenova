package carrierintelservice

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const ensureFreshConcurrency = 4

func isVersionConflict(err error) bool {
	return errortypes.IsVersionMismatchError(err)
}

func (s *Service) GateFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (map[pulid.ID]*carrier.IntelGate, error) {
	gates := make(map[pulid.ID]*carrier.IntelGate, len(carrierIDs))
	if len(carrierIDs) == 0 {
		return gates, nil
	}

	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	primary, ok := control.PrimaryType()
	if !ok {
		return gates, nil
	}

	snapshots, err := s.snapshotRepo.GetCurrentByCarrierIDs(ctx, tenantInfo, carrierIDs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	overrides, err := s.overrideRepo.ListActiveByCarrierIDs(ctx, tenantInfo, carrierIDs, now)
	if err != nil {
		return nil, err
	}

	overridesByCarrier := make(map[pulid.ID][]*carrierintel.CarrierIntelOverride, len(overrides))
	for _, o := range overrides {
		overridesByCarrier[o.CarrierID] = append(overridesByCarrier[o.CarrierID], o)
	}
	snapshotByCarrier := make(map[pulid.ID]*carrierintel.CarrierIntelSnapshot, len(snapshots))
	for _, snap := range snapshots {
		snapshotByCarrier[snap.CarrierID] = snap
	}

	unavailable := s.breaker.isOpen(breakerKey(tenantInfo, primary), time.Unix(now, 0))
	outage := carrier.IntelOutagePolicy(control.OutagePolicy)

	needsRecompute := make([]pulid.ID, 0, len(snapshots))
	for _, snap := range snapshots {
		if snap.PolicyVersion != control.PolicyVersion ||
			len(overridesByCarrier[snap.CarrierID]) > 0 ||
			hasOverriddenFinding(snap) {
			needsRecompute = append(needsRecompute, snap.CarrierID)
		}
	}
	subjectByCarrier := make(map[pulid.ID]repositories.CarrierIntelSubject, len(needsRecompute))
	if len(needsRecompute) > 0 {
		subjects, subjectErr := s.subjectRepo.ListCarrierSubjects(
			ctx,
			&repositories.ListCarrierIntelSubjectsRequest{
				TenantInfo: tenantInfo,
				CarrierIDs: needsRecompute,
				Limit:      len(needsRecompute),
			},
		)
		if subjectErr != nil {
			return nil, subjectErr
		}
		for _, subject := range subjects {
			subjectByCarrier[subject.CarrierID] = subject
		}
	}

	for _, carrierID := range carrierIDs {
		snap := snapshotByCarrier[carrierID]
		gate := &carrier.IntelGate{
			Evaluated:           true,
			Provider:            primary.String(),
			ProviderUnavailable: unavailable,
			OutagePolicy:        outage,
			Findings:            []carrier.IntelFinding{},
		}
		if snap == nil {
			gate.Stale = true
			gate.PastHardMaxAge = true
			gates[carrierID] = gate
			continue
		}

		age := now - snap.EffectiveAsOf()
		gate.FetchedAt = snap.EffectiveAsOf()
		gate.NotFound = snap.NotFound
		gate.Stale = age > control.PreTenderMaxAgeSeconds()
		gate.PastHardMaxAge = age > control.HardMaxAgeSeconds()

		findings := snap.Findings
		if subject, ok := subjectByCarrier[carrierID]; ok {
			findings = carrierintel.CarryUnconfirmed(snap.Findings, carrierintel.EvaluateFindings(
				&carrierintel.EvaluateInput{
					Profile:  snap.Profile,
					Now:      now,
					Subject:  carrierintel.SubjectTypeCarrier,
					NotFound: snap.NotFound,
					Settings: control.Rules,
					Overrides: carrierintel.OverrideRefs(
						overridesByCarrier[carrierID], now,
					),
					BrokerAuthority: subject.Broker,
					AuthorityExempt: subject.Exempt,
				},
			))
		}
		gate.Findings = toGateFindings(findings)
		gates[carrierID] = gate
	}

	return gates, nil
}

func hasOverriddenFinding(snap *carrierintel.CarrierIntelSnapshot) bool {
	for idx := range snap.Findings {
		if snap.Findings[idx].Overridden {
			return true
		}
	}
	return false
}

func toGateFindings(findings []carrierintel.Finding) []carrier.IntelFinding {
	out := make([]carrier.IntelFinding, 0, len(findings))
	for idx := range findings {
		finding := findings[idx]
		if finding.Unverifiable || finding.Action == carrierintel.RuleActionOff {
			continue
		}
		out = append(out, carrier.IntelFinding{
			Code:              finding.Code.String(),
			Category:          finding.Category.String(),
			Action:            carrier.IntelAction(finding.Action),
			Message:           finding.Message,
			Overridden:        finding.Overridden,
			OverrideExpiresAt: finding.OverrideExpiresAt,
		})
	}
	return out
}

func (s *Service) EnsureFresh(ctx context.Context, req *services.CarrierIntelEnsureFreshRequest) {
	if len(req.CarrierIDs) == 0 {
		return
	}
	control, err := s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		s.l.Warn("failed to load carrier intelligence control", zap.Error(err))
		return
	}
	if _, ok := control.PrimaryType(); !ok || !control.PreTenderRefreshEnabled {
		return
	}

	snapshots, err := s.snapshotRepo.GetCurrentByCarrierIDs(ctx, req.TenantInfo, req.CarrierIDs)
	if err != nil {
		s.l.Warn("failed to load carrier intelligence snapshots", zap.Error(err))
		return
	}
	fresh := make(map[pulid.ID]bool, len(snapshots))
	now := s.now()
	for _, snap := range snapshots {
		if snap.IsFresh(now, control.PreTenderMaxAgeSeconds()) {
			fresh[snap.CarrierID] = true
		}
	}

	stale := make([]pulid.ID, 0, len(req.CarrierIDs))
	for _, id := range req.CarrierIDs {
		if !fresh[id] {
			stale = append(stale, id)
		}
	}
	if len(stale) == 0 {
		return
	}

	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&repositories.ListCarrierIntelSubjectsRequest{
			TenantInfo: req.TenantInfo,
			CarrierIDs: stale,
			Limit:      len(stale),
		},
	)
	if err != nil {
		s.l.Warn("failed to load carrier subjects for refresh", zap.Error(err))
		return
	}

	budget := req.Budget
	if budget <= 0 {
		budget = s.interactiveWait
	}
	refreshCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	purpose := req.Purpose
	if !purpose.IsValid() {
		purpose = carrierintel.PurposePreTender
	}

	sem := make(chan struct{}, ensureFreshConcurrency)
	var wg sync.WaitGroup
	for _, subject := range subjects {
		wg.Go(func() {
			select {
			case sem <- struct{}{}:
			case <-refreshCtx.Done():
				return
			}
			defer func() { <-sem }()

			if _, fetchErr := s.Fetch(refreshCtx, &FetchRequest{
				TenantInfo:    req.TenantInfo,
				Subject:       subject,
				MinDepth:      carrierintel.LookupDepthFMCSA,
				MaxAgeSeconds: control.PreTenderMaxAgeSeconds(),
				Purpose:       purpose,
				Control:       control,
			}); fetchErr != nil {
				s.l.Info("pre-tender carrier intelligence refresh failed",
					zap.String("carrierId", subject.CarrierID.String()), zap.Error(fetchErr))
			}
		})
	}
	wg.Wait()
}
