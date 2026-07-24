package telematicsservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type TenantSweepResult struct {
	VehiclesMatched    int `json:"vehiclesMatched"`
	VehiclesUnmatched  int `json:"vehiclesUnmatched"`
	TrailersMatched    int `json:"trailersMatched"`
	ViolationsUpserted int `json:"violationsUpserted"`
	RulesetsUpdated    int `json:"rulesetsUpdated"`
	DVIRsUpserted      int `json:"dvirsUpserted"`
	FormsUpserted      int `json:"formsUpserted"`
}

func (s *Service) SweepTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*TenantSweepResult, error) {
	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	result := new(TenantSweepResult)
	mapErr := s.syncVehicleMappings(ctx, tenantInfo, provider, result)
	trailerErr := s.syncTrailerMappings(ctx, tenantInfo, provider, result)
	rulesetErr := s.syncDriverRulesets(ctx, tenantInfo, provider, result)
	violationErr := s.pollHOSViolations(ctx, tenantInfo, provider, result)
	dvirErr := s.syncDVIRs(ctx, tenantInfo, provider, result)
	formErr := s.syncForms(ctx, tenantInfo, provider, result)
	return result, errors.Join(
		mapErr,
		trailerErr,
		rulesetErr,
		violationErr,
		dvirErr,
		formErr,
	)
}

type unitMappingCandidate struct {
	unitID     pulid.ID
	externalID string
	vin        string
	code       string
}

type unitExternalIDMatch struct {
	unitID     pulid.ID
	externalID string
}

func matchUnitsToProviderVehicles(
	candidates []unitMappingCandidate,
	vehicles []services.ProviderVehicle,
	unmatched *int,
) []unitExternalIDMatch {
	assignedExternalIDs := make(map[string]struct{}, len(candidates))
	unmappedByVin := make(map[string]pulid.ID)
	unmappedByCode := make(map[string]pulid.ID)
	for _, candidate := range candidates {
		if candidate.externalID != "" {
			assignedExternalIDs[candidate.externalID] = struct{}{}
			continue
		}
		if vin := strings.ToUpper(strings.TrimSpace(candidate.vin)); vin != "" {
			unmappedByVin[vin] = candidate.unitID
		}
		if code := strings.ToLower(strings.TrimSpace(candidate.code)); code != "" {
			unmappedByCode[code] = candidate.unitID
		}
	}

	if len(unmappedByVin) == 0 && len(unmappedByCode) == 0 {
		return nil
	}

	matches := make([]unitExternalIDMatch, 0)
	claimedUnits := make(map[pulid.ID]struct{})
	for i := range vehicles {
		vehicle := &vehicles[i]
		if _, taken := assignedExternalIDs[vehicle.ID]; taken {
			continue
		}

		unitID, matched := matchVehicleToUnit(vehicle, unmappedByVin, unmappedByCode)
		if !matched {
			if unmatched != nil {
				*unmatched++
			}
			continue
		}
		if _, claimed := claimedUnits[unitID]; claimed {
			continue
		}
		claimedUnits[unitID] = struct{}{}
		assignedExternalIDs[vehicle.ID] = struct{}{}
		matches = append(matches, unitExternalIDMatch{
			unitID:     unitID,
			externalID: vehicle.ID,
		})
	}
	return matches
}

func matchVehicleToUnit(
	vehicle *services.ProviderVehicle,
	unmappedByVin map[string]pulid.ID,
	unmappedByCode map[string]pulid.ID,
) (pulid.ID, bool) {
	if vin := strings.ToUpper(strings.TrimSpace(vehicle.VIN)); vin != "" {
		if unitID, ok := unmappedByVin[vin]; ok {
			return unitID, true
		}
	}
	if name := strings.ToLower(strings.TrimSpace(vehicle.Name)); name != "" {
		if unitID, ok := unmappedByCode[name]; ok {
			return unitID, true
		}
	}
	return pulid.Nil, false
}

func (s *Service) syncVehicleMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantSweepResult,
) error {
	mappings, err := s.repo.ListTractorMappings(ctx, tenantInfo)
	if err != nil {
		return err
	}

	candidates := make([]unitMappingCandidate, 0, len(mappings))
	unmapped := 0
	for _, mapping := range mappings {
		if mapping.ExternalID == "" {
			unmapped++
		}
		candidates = append(candidates, unitMappingCandidate{
			unitID:     mapping.TractorID,
			externalID: mapping.ExternalID,
			vin:        mapping.Vin,
			code:       mapping.Code,
		})
	}
	if unmapped == 0 {
		return nil
	}

	providerVehicles, err := provider.ListVehicles(ctx)
	if err != nil {
		return err
	}

	matches := matchUnitsToProviderVehicles(
		candidates,
		providerVehicles,
		&result.VehiclesUnmatched,
	)
	if len(matches) == 0 {
		return nil
	}

	assignments := make([]repositories.TractorExternalIDAssignment, 0, len(matches))
	for _, match := range matches {
		assignments = append(assignments, repositories.TractorExternalIDAssignment{
			TractorID:  match.unitID,
			ExternalID: match.externalID,
		})
	}

	assigned, err := s.repo.AssignTractorExternalIDs(
		ctx,
		repositories.AssignTractorExternalIDsRequest{
			TenantInfo:  tenantInfo,
			Assignments: assignments,
		},
	)
	if err != nil {
		return err
	}
	result.VehiclesMatched = assigned

	s.l.Info("assigned telematics vehicle mappings",
		zap.String("organizationId", tenantInfo.OrgID.String()),
		zap.String("provider", string(provider.Type())),
		zap.Int("assigned", assigned))
	return nil
}

func (s *Service) syncTrailerMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantSweepResult,
) error {
	mappings, err := s.repo.ListTrailerMappings(ctx, tenantInfo)
	if err != nil {
		return err
	}

	candidates := make([]unitMappingCandidate, 0, len(mappings))
	unmapped := 0
	for _, mapping := range mappings {
		if mapping.ExternalID == "" {
			unmapped++
		}
		candidates = append(candidates, unitMappingCandidate{
			unitID:     mapping.TrailerID,
			externalID: mapping.ExternalID,
			vin:        mapping.Vin,
			code:       mapping.Code,
		})
	}
	if unmapped == 0 {
		return nil
	}

	providerTrailers, err := provider.ListTrailers(ctx)
	if err != nil {
		return err
	}

	matches := matchUnitsToProviderVehicles(candidates, providerTrailers, nil)
	if len(matches) == 0 {
		return nil
	}

	assignments := make([]repositories.TrailerExternalIDAssignment, 0, len(matches))
	for _, match := range matches {
		assignments = append(assignments, repositories.TrailerExternalIDAssignment{
			TrailerID:  match.unitID,
			ExternalID: match.externalID,
		})
	}

	assigned, err := s.repo.AssignTrailerExternalIDs(
		ctx,
		repositories.AssignTrailerExternalIDsRequest{
			TenantInfo:  tenantInfo,
			Assignments: assignments,
		},
	)
	if err != nil {
		return err
	}
	result.TrailersMatched = assigned

	s.l.Info("assigned telematics trailer mappings",
		zap.String("organizationId", tenantInfo.OrgID.String()),
		zap.String("provider", string(provider.Type())),
		zap.Int("assigned", assigned))
	return nil
}

func (s *Service) syncDriverRulesets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantSweepResult,
) error {
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if len(workersByExternalID) == 0 {
		return nil
	}

	profiles, err := provider.ListDriverProfiles(ctx)
	if err != nil {
		return err
	}

	assignments := make([]repositories.WorkerRulesetAssignment, 0, len(profiles))
	for i := range profiles {
		profile := &profiles[i]
		if profile.Ruleset == nil {
			continue
		}
		workerID, ok := workersByExternalID[profile.DriverID]
		if !ok {
			continue
		}
		assignments = append(assignments, repositories.WorkerRulesetAssignment{
			WorkerID:     workerID,
			Cycle:        profile.Ruleset.Cycle,
			Shift:        profile.Ruleset.Shift,
			Restart:      profile.Ruleset.Restart,
			Break:        profile.Ruleset.Break,
			Jurisdiction: profile.Ruleset.Jurisdiction,
		})
	}

	if len(assignments) == 0 {
		return nil
	}

	updated, err := s.repo.UpdateWorkerRulesets(ctx, repositories.UpdateWorkerRulesetsRequest{
		TenantInfo:  tenantInfo,
		Assignments: assignments,
	})
	if err != nil {
		return err
	}
	result.RulesetsUpdated = updated
	return nil
}

func (s *Service) pollHOSViolations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantSweepResult,
) error {
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if len(workersByExternalID) == 0 {
		return nil
	}

	now := timeutils.NowUnix()
	providerViolations, err := provider.ListHOSViolations(
		ctx,
		now-violationLookbackSeconds,
		now,
	)
	if err != nil {
		return err
	}

	violations := make([]*telematics.WorkerHOSViolation, 0, len(providerViolations))
	for i := range providerViolations {
		violation := &providerViolations[i]
		workerID, ok := workersByExternalID[violation.DriverID]
		if !ok {
			continue
		}
		violations = append(violations, &telematics.WorkerHOSViolation{
			OrganizationID:   tenantInfo.OrgID,
			BusinessUnitID:   tenantInfo.BuID,
			WorkerID:         workerID,
			ViolationType:    violation.Type,
			ViolationStartAt: violation.StartAt,
			Description:      violation.Description,
			DurationMs:       violation.DurationMs,
			DayStartAt:       violation.DayStartAt,
			DayEndAt:         violation.DayEndAt,
			DetectedAt:       now,
		})
	}

	if err = s.repo.UpsertWorkerHOSViolations(ctx, violations); err != nil {
		return err
	}
	result.ViolationsUpserted = len(violations)

	if len(violations) > 0 {
		s.publishInvalidation(ctx, tenantInfo, "workerHosViolation")
	}
	return nil
}

func (s *Service) syncDVIRs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantSweepResult,
) error {
	tractorsByExternalID, err := s.tractorsByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	records, err := provider.ListDVIRs(ctx, now-dvirLookbackSeconds, now)
	if err != nil {
		return err
	}

	providerType := string(provider.Type())
	inspections := make([]*telematics.VehicleInspection, 0, len(records))
	for i := range records {
		record := &records[i]
		if record.ID == "" {
			continue
		}
		inspection := &telematics.VehicleInspection{
			ID:             telematics.NewVehicleInspectionID(),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Provider:       providerType,
			ProviderDvirID: record.ID,
			InspectionType: record.Type,
			SafetyStatus:   record.SafetyStatus,
			StartedAt:      record.StartAt,
			EndedAt:        record.EndAt,
			OdometerMeters: record.OdometerMeters,
			Location:       record.Location,
			Signed:         record.Signed,
			CreatedAt:      now,
		}
		if tractorID, ok := tractorsByExternalID[record.VehicleID]; ok {
			inspection.TractorID = tractorID
		}
		if workerID, ok := workersByExternalID[record.DriverID]; ok {
			inspection.WorkerID = workerID
		}
		applyInspectionDefects(inspection, record)
		inspections = append(inspections, inspection)
	}

	if err = s.repo.UpsertVehicleInspections(ctx, inspections); err != nil {
		return err
	}
	result.DVIRsUpserted = len(inspections)

	if len(inspections) > 0 {
		s.publishInvalidation(ctx, tenantInfo, "vehicleInspection")
	}
	return nil
}

func applyInspectionDefects(
	inspection *telematics.VehicleInspection,
	record *services.ProviderDVIR,
) {
	inspection.DefectCount = len(record.Defects)
	if len(record.Defects) == 0 {
		return
	}

	defects := make([]telematics.VehicleInspectionDefect, 0, len(record.Defects))
	unresolved := 0
	for i := range record.Defects {
		defect := &record.Defects[i]
		if !defect.Resolved {
			unresolved++
		}
		defects = append(defects, telematics.VehicleInspectionDefect{
			ID:         defect.ID,
			DefectType: defect.DefectType,
			Comment:    defect.Comment,
			Resolved:   defect.Resolved,
			ResolvedAt: defect.ResolvedAt,
		})
	}
	if inspection.SafetyStatus == dvirSafetyStatusResolved {
		unresolved = 0
	}
	inspection.UnresolvedDefectCount = unresolved
	inspection.Defects = defects
}
