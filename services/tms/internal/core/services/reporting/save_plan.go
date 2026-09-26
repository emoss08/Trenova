package reporting

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

func NewDefinition(req *SaveDefinitionRequest) *report.ReportDefinition {
	return &report.ReportDefinition{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		Kind:           report.DefinitionKindCustom,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     defaultVisibility(req.Visibility),
		Status:         defaultStatus(req.Status),
		CatalogVersion: reportcatalog.Version,
		Definition:     req.Definition,
		DefaultFormat:  defaultFormat(req.DefaultFormat),
	}
}

func ApplyDefinitionSave(existing *report.ReportDefinition, req *SaveDefinitionRequest) {
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = defaultVisibility(req.Visibility)
	existing.Status = defaultStatus(req.Status)
	existing.Definition = req.Definition
	existing.CatalogVersion = reportcatalog.Version
	existing.DefaultFormat = defaultFormat(req.DefaultFormat)
	existing.Diagnostics = nil
	existing.Version = req.Version
}

func NewCannedFork(entry *canned.Entry, req *ForkCannedRequest) *report.ReportDefinition {
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
		CatalogVersion: reportcatalog.Version,
		Definition:     entry.Definition,
		DefaultFormat:  entry.DefaultFormat,
	}
}

func NewDashboard(req *SaveDashboardRequest) *report.Dashboard {
	return &report.Dashboard{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     defaultVisibility(req.Visibility),
		Layout:         defaultLayout(req.Layout),
	}
}

func ApplyDashboardSave(existing *report.Dashboard, req *SaveDashboardRequest) {
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = defaultVisibility(req.Visibility)
	existing.Layout = defaultLayout(req.Layout)
	existing.Version = req.Version
}
