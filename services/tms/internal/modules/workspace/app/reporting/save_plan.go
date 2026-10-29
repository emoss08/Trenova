package reporting

import (
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

func (s *Service) NewDefinition(req *SaveDefinitionRequest) *report.ReportDefinition {
	return NewDefinition(req, s.CatalogVersion())
}

func (s *Service) ApplyDefinitionSave(
	existing *report.ReportDefinition,
	req *SaveDefinitionRequest,
) {
	ApplyDefinitionSave(existing, req, s.CatalogVersion())
}

func (s *Service) NewCannedFork(
	entry *canned.Entry,
	req *ForkCannedRequest,
) *report.ReportDefinition {
	return NewCannedFork(entry, req, s.CatalogVersion())
}

func NewDefinition(req *SaveDefinitionRequest, catalogVersion string) *report.ReportDefinition {
	return &report.ReportDefinition{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		Kind:           report.DefinitionKindCustom,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     DefaultVisibility(req.Visibility),
		Status:         defaultStatus(req.Status),
		CatalogVersion: catalogVersion,
		Definition:     req.Definition,
		DefaultFormat:  defaultFormat(req.DefaultFormat),
	}
}

func ApplyDefinitionSave(
	existing *report.ReportDefinition,
	req *SaveDefinitionRequest,
	catalogVersion string,
) {
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = DefaultVisibility(req.Visibility)
	existing.Status = defaultStatus(req.Status)
	existing.Definition = req.Definition
	existing.CatalogVersion = catalogVersion
	existing.DefaultFormat = defaultFormat(req.DefaultFormat)
	existing.Diagnostics = nil
	existing.Version = req.Version
}

func NewCannedFork(
	entry *canned.Entry,
	req *ForkCannedRequest,
	catalogVersion string,
) *report.ReportDefinition {
	name := req.Name
	if name == "" {
		name = entry.Name
	}

	return &report.ReportDefinition{
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
		CatalogVersion: catalogVersion,
		Definition:     entry.Definition,
		DefaultFormat:  entry.DefaultFormat,
	}
}
