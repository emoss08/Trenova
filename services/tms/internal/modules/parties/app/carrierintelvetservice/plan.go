package carrierintelvetservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsync"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// VetPlan is what vetting would ask the provider for: the subject and the
// provider that answers, with nothing fetched.
type VetPlan struct {
	Subject  carrierintelports.CarrierIntelSubject
	Provider integration.Type
	Depth    carrierintel.LookupDepth
	Current  *carrierintelsnapshot.CarrierIntelSnapshot
}

func (s *Service) PlanVetCarrier(ctx context.Context, req *VetCarrierRequest) (*VetPlan, error) {
	subject, err := s.loadCarrierSubject(ctx, req.TenantInfo, req.CarrierID)
	if err != nil {
		return nil, err
	}
	depth := req.Depth
	if !depth.IsValid() {
		depth = carrierintel.LookupDepthFull
	}

	return s.planFetch(ctx, req.TenantInfo, &subject, depth)
}

func (s *Service) PlanVetCustomerBroker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*VetPlan, error) {
	subject, err := s.subjectRepo.GetCustomerSubject(ctx, tenantInfo, customerID)
	if err != nil {
		return nil, err
	}
	if subject.DOTNumber == "" {
		return nil, errortypes.NewBusinessError(
			"Add the customer's DOT number before vetting them as a broker",
		)
	}
	subject.Broker = true

	return s.planFetch(ctx, tenantInfo, subject, carrierintel.LookupDepthFMCSA)
}

func (s *Service) planFetch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject *carrierintelports.CarrierIntelSubject,
	depth carrierintel.LookupDepth,
) (*VetPlan, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before vetting carriers",
		)
	}

	current, err := s.snapshotRepo.GetCurrent(
		ctx,
		tenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: subject.SubjectType,
			SubjectID:   subject.SubjectID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &VetPlan{Subject: *subject, Provider: provider, Depth: depth, Current: current}, nil
}

// SuggestionPlan is the carrier as applying the chosen suggestions would
// leave it, and the suggestions that still apply.
type SuggestionPlan struct {
	Before    *carrier.Carrier
	After     *carrier.Carrier
	Fields    []carrierintelsync.FieldUpdate
	Insurance []carrierintelsync.InsuranceChange
	Applied   int
}

func (s *Service) PlanApplySuggestions(
	ctx context.Context,
	req *ApplySuggestionsRequest,
) (*SuggestionPlan, error) {
	if len(req.Fields) == 0 && len(req.PolicyIDs) == 0 {
		return nil, errortypes.NewValidationError("fields", errortypes.ErrRequired,
			"Select at least one suggestion to apply")
	}
	for _, field := range req.Fields {
		if !field.IsValid() {
			return nil, errortypes.NewValidationError("fields", errortypes.ErrInvalid,
				"Suggestion field is invalid")
		}
	}

	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		req.TenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   req.CarrierID.String(),
		},
	)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.NotFound {
		return nil, errortypes.NewBusinessError("Run carrier intelligence for this carrier first")
	}

	load := func() (*carrier.Carrier, error) {
		return s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
			ID:         req.CarrierID,
			TenantInfo: req.TenantInfo,
			CarrierFilterOptions: carrierports.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
				IncludeEDIChannels:       true,
			},
		})
	}
	before, err := load()
	if err != nil {
		return nil, err
	}
	entity, err := load()
	if err != nil {
		return nil, err
	}

	plan := carrierintelsync.PlanCarrierSync(
		entity,
		snapshot.Profile,
		carrierintelsync.SyncSettings{},
		s.now(),
	)
	selected := make([]carrierintelsync.FieldUpdate, 0, len(req.Fields))
	for idx := range plan.Suggestions {
		if slices.Contains(req.Fields, plan.Suggestions[idx].Field) {
			selected = append(selected, plan.Suggestions[idx])
		}
	}
	selectedInsurance := make([]carrierintelsync.InsuranceChange, 0, len(req.PolicyIDs))
	for idx := range plan.InsuranceSuggestions {
		change := &plan.InsuranceSuggestions[idx]
		if change.PolicyID.IsNotNil() && slices.Contains(req.PolicyIDs, change.PolicyID) {
			selectedInsurance = append(selectedInsurance, *change)
		}
	}
	if len(selected) == 0 && len(selectedInsurance) == 0 {
		return nil, errortypes.NewBusinessError(
			"The selected suggestions no longer apply. Refresh the carrier and try again",
		)
	}

	applied := len(carrierintelsync.ApplyFieldUpdates(entity, selected))
	applied += carrierintelsync.ApplyInsuranceChanges(entity, selectedInsurance)

	return &SuggestionPlan{
		Before:    before,
		After:     entity,
		Fields:    selected,
		Insurance: selectedInsurance,
		Applied:   applied,
	}, nil
}
