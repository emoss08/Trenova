package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"go.uber.org/zap"
)

func (s *Service) ListCanned() []*canned.Entry {
	return s.canned.All()
}

func (s *Service) GetCanned(key string) (*canned.Entry, error) {
	entry, ok := s.canned.Get(key)
	if !ok {
		return nil, errortypes.NewNotFoundError(
			fmt.Sprintf("Canned report %q not found", key),
		)
	}
	return entry, nil
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

	name := req.Name
	if name == "" {
		name = entry.Name
	}

	entity := &report.ReportDefinition{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           name,
		Description:    entry.Description,
		Category:       entry.Category,
		Tags:           entry.Tags,
		Kind:           report.DefinitionKindCannedFork,
		CannedKey:      entry.Key,
		CannedVersion:  entry.Version,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     report.VisibilityPrivate,
		Status:         report.DefinitionStatusActive,
		CatalogVersion: reportcatalog.Version,
		Definition:     entry.Definition,
		DefaultFormat:  entry.DefaultFormat,
	}

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
	existing, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the report owner can reset this report",
		)
	}
	if existing.Kind != report.DefinitionKindCannedFork || existing.CannedKey == "" {
		return nil, errortypes.NewBusinessError(
			"Only reports customized from a canned report can be reset",
		)
	}

	entry, err := s.GetCanned(existing.CannedKey)
	if err != nil {
		return nil, err
	}

	existing.Definition = entry.Definition
	existing.CannedVersion = entry.Version
	existing.CatalogVersion = reportcatalog.Version
	existing.Status = report.DefinitionStatusActive
	existing.Diagnostics = nil

	return s.defRepo.Update(ctx, existing, req.TenantInfo.UserID)
}
