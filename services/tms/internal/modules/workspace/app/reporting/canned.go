package reporting

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type CannedReport struct {
	Entry     *canned.Entry
	Available bool
	Plan      *permissionports.PlanDenial
}

type cannedReaches struct {
	once  sync.Once
	byKey map[string]*reportaccess.Reach
}

var cannedPlanMessage = errortypes.BaseError{
	Message: "\"{0}\" is part of the {1} add-on, which is not on your plan",
}

func (s *Service) ListCanned() []*canned.Entry {
	return s.canned.All()
}

func (s *Service) LookupCanned(key string) (*canned.Entry, bool) {
	return s.canned.Get(key)
}

func (s *Service) GetCanned(key string) (*canned.Entry, error) {
	entry, ok := s.LookupCanned(key)
	if !ok {
		return nil, errortypes.NewNotFoundError(
			"Canned report \"{0}\" not found", key,
		)
	}
	return entry, nil
}

func (s *Service) CannedLibrary(ctx context.Context, req *Request) ([]CannedReport, error) {
	access, err := s.access.For(ctx, req.TenantInfo, reportaccess.ModeRead)
	if err != nil {
		return nil, err
	}

	entries := s.canned.All()
	out := make([]CannedReport, 0, len(entries))
	for _, entry := range entries {
		denial := s.cannedPlan(ctx, access, entry)
		out = append(out, CannedReport{Entry: entry, Available: denial == nil, Plan: denial})
	}
	return out, nil
}

func (s *Service) RequireCannedOnPlan(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entry *canned.Entry,
) error {
	access, err := s.access.For(ctx, tenant, reportaccess.ModeRead)
	if err != nil {
		return err
	}
	denial := s.cannedPlan(ctx, access, entry)
	if denial == nil {
		return nil
	}
	restriction := denial.Restriction()
	if denial.AddOnName != "" {
		restriction.Message = cannedPlanMessage.Message
		restriction.Args = []any{entry.Name, denial.AddOnName}
	}
	return restriction
}

func (s *Service) cannedPlan(
	ctx context.Context,
	access *reportaccess.Access,
	entry *canned.Entry,
) *permissionports.PlanDenial {
	reach, ok := s.cannedReach(entry.Key)
	if !ok {
		return nil
	}
	return access.OffPlan(ctx, reach)
}

func (s *Service) cannedReach(key string) (*reportaccess.Reach, bool) {
	s.reaches.once.Do(func() {
		entries := s.canned.All()
		s.reaches.byKey = make(map[string]*reportaccess.Reach, len(entries))
		for _, entry := range entries {
			reach, err := s.access.ReachOf(entry.Definition)
			if err != nil {
				s.l.Warn("canned report does not resolve against the catalog",
					zap.String("cannedKey", entry.Key), zap.Error(err))
				continue
			}
			s.reaches.byKey[entry.Key] = reach
		}
	})
	reach, ok := s.reaches.byKey[key]
	return reach, ok
}

type ForkCannedRequest struct {
	Request

	CannedKey string
	Name      string
}

func (s *Service) ForkCanned(
	ctx context.Context,
	req *ForkCannedRequest,
) (*report.ReportDefinition, error) {
	log := s.l.With(zap.String("operation", "ForkCanned"), zap.String("cannedKey", req.CannedKey))

	entry, err := s.GetCanned(req.CannedKey)
	if err != nil {
		return nil, err
	}
	if err = s.RequireCannedOnPlan(ctx, req.TenantInfo, entry); err != nil {
		return nil, err
	}

	entity := s.NewCannedFork(entry, req)

	created, err := s.defRepo.Create(ctx, entity, req.TenantInfo.UserID)
	if err != nil {
		log.Error("failed to fork canned report", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) ResetCannedFork(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	change, err := s.PlanResetCannedFork(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.defRepo.Update(ctx, change.After, req.TenantInfo.UserID)
}
