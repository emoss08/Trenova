package carrierintelvetservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsync"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type VetCarrierRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierID  pulid.ID
	Depth      carrierintel.LookupDepth
	Force      bool
}

func (s *Service) loadCarrierSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (carrierintelports.CarrierIntelSubject, error) {
	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&carrierintelports.ListCarrierIntelSubjectsRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: []pulid.ID{carrierID},
			Limit:      1,
		},
	)
	if err != nil {
		return carrierintelports.CarrierIntelSubject{}, err
	}
	if len(subjects) == 0 {
		if _, getErr := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
			ID:         carrierID,
			TenantInfo: tenantInfo,
		}); getErr != nil {
			return carrierintelports.CarrierIntelSubject{}, getErr
		}
		return carrierintelports.CarrierIntelSubject{}, errortypes.NewBusinessError(
			"Add the carrier's DOT number before running carrier intelligence",
		)
	}
	return subjects[0], nil
}

func (s *Service) VetCarrier(
	ctx context.Context,
	req *VetCarrierRequest,
) (*carrierintelservice.FetchResult, error) {
	subject, err := s.loadCarrierSubject(ctx, req.TenantInfo, req.CarrierID)
	if err != nil {
		return nil, err
	}
	purpose := carrierintel.PurposeVet
	depth := req.Depth
	if depth.IsValid() && depth != carrierintel.LookupDepthFull {
		purpose = carrierintel.PurposeRefresh
	}
	return s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
		TenantInfo: req.TenantInfo,
		Subject:    subject,
		MinDepth:   depth,
		Purpose:    purpose,
		Force:      req.Force,
		Source:     carrierintelevent.EventSourceSnapshotDiff,
	})
}

type SelfIntelligenceResult struct {
	Configured bool
	DOTNumber  string
	Snapshot   *carrierintelsnapshot.CarrierIntelSnapshot
}

func (s *Service) MyIntelligence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	refresh bool,
) (*SelfIntelligenceResult, error) {
	org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if org == nil || org.DOTNumber == "" {
		return &SelfIntelligenceResult{}, nil
	}
	result := &SelfIntelligenceResult{Configured: true, DOTNumber: org.DOTNumber}
	if !refresh {
		snapshot, getErr := s.snapshotRepo.GetCurrent(
			ctx,
			tenantInfo,
			carrierintelports.CarrierIntelSubjectRef{
				SubjectType: carrierintel.SubjectTypeOrganization,
				SubjectID:   org.SubjectID,
			},
		)
		if getErr != nil {
			return nil, getErr
		}
		result.Snapshot = snapshot
		return result, nil
	}

	fetched, err := s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
		TenantInfo: tenantInfo,
		Subject:    *org,
		Purpose:    carrierintel.PurposeSelfMonitor,
		Force:      true,
		Source:     carrierintelevent.EventSourceSnapshotDiff,
	})
	if err != nil {
		return nil, err
	}
	result.Snapshot = fetched.Snapshot
	return result, nil
}

func (s *Service) VetCustomerBroker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	force bool,
) (*carrierintelservice.FetchResult, error) {
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
	return s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
		TenantInfo: tenantInfo,
		Subject:    *subject,
		MinDepth:   carrierintel.LookupDepthFMCSA,
		Purpose:    carrierintel.PurposeVet,
		Force:      force,
		Source:     carrierintelevent.EventSourceSnapshotDiff,
	})
}

func (s *Service) SyncPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (*carrierintelsync.SyncPlan, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	entity, err := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		tenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   carrierID.String(),
		},
	)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.NotFound {
		empty := carrierintelsync.PlanCarrierSync(nil, nil, control.SyncSettings(), s.now())
		return &empty, nil
	}
	plan := carrierintelsync.PlanCarrierSync(
		entity,
		snapshot.Profile,
		carrierintelsync.SyncSettings{},
		s.now(),
	)
	return &plan, nil
}

type ApplySuggestionsRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierID  pulid.ID
	Fields     []carrierintelsync.SyncField
	PolicyIDs  []pulid.ID
}

func (s *Service) ApplySuggestions(
	ctx context.Context,
	req *ApplySuggestionsRequest,
) (int, error) {
	plan, err := s.PlanApplySuggestions(ctx, req)
	if err != nil {
		return 0, err
	}
	entity := plan.After

	if _, err = s.carrierService.Update(ctx, entity, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}); err != nil {
		return 0, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    req.CarrierID.String(),
		Operation:     permission.OpUpdate,
		UserID:        req.TenantInfo.UserID,
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   req.TenantInfo.UserID,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"fields":    plan.Fields,
			"insurance": plan.Insurance,
		}),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, auditservice.WithComment("Carrier intelligence suggestions applied")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
	return plan.Applied, nil
}
