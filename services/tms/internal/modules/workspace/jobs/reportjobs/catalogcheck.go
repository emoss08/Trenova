package reportjobs

import (
	"context"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

const (
	catalogCheckBatch     = 200
	catalogCheckMaxPerRun = 20_000
	missingDefinition     = "definition: the report has no definition"
	catalogCheckEvent     = "catalog_check"
	dataKeyCatalog        = "catalogVersion"
	dataKeyDiagnostics    = "diagnostics"
	dataKeyCatalogEvent   = "event"
)

type CheckCatalogResult struct {
	Checked        int `json:"checked"`
	NeedsAttention int `json:"needsAttention"`
	Restored       int `json:"restored"`
}

func (a *Activities) CheckDefinitionsAgainstCatalogActivity(
	ctx context.Context,
) (*CheckCatalogResult, error) {
	result := &CheckCatalogResult{}
	version := a.engine.Catalog().Version()

	for result.Checked < catalogCheckMaxPerRun {
		stale, err := a.defRepo.ListCatalogStale(
			ctx,
			&reportports.ListCatalogStaleReportDefinitionsRequest{
				CatalogVersion: version,
				Limit:          catalogCheckBatch,
			},
		)
		if err != nil {
			return nil, err
		}

		settled := 0
		for _, def := range stale {
			activity.RecordHeartbeat(ctx, def.ID.String())
			if a.checkAgainstCatalog(ctx, def, version, result) {
				settled++
			}
		}
		result.Checked += settled

		if len(stale) < catalogCheckBatch || settled == 0 {
			break
		}
	}

	if result.NeedsAttention > 0 || result.Restored > 0 {
		a.l.Info("checked saved reports against the report catalog",
			zap.String("catalogVersion", version),
			zap.Int("checked", result.Checked),
			zap.Int("needsAttention", result.NeedsAttention),
			zap.Int("restored", result.Restored))
	}

	return result, nil
}

func (a *Activities) checkAgainstCatalog(
	ctx context.Context,
	def *report.ReportDefinition,
	version string,
	result *CheckCatalogResult,
) bool {
	diagnostics := a.catalogDiagnostics(def.Definition)
	status := catalogCheckStatus(def.Status, diagnostics)
	tenant := pagination.TenantInfo{OrgID: def.OrganizationID, BuID: def.BusinessUnitID}

	recorded, err := a.defRepo.RecordCatalogCheck(
		dbscope.WithTenant(ctx, tenant.DBTenant()),
		&reportports.RecordReportCatalogCheckRequest{
			TenantInfo:     tenant,
			DefinitionID:   def.ID,
			Version:        def.Version,
			CatalogVersion: version,
			Status:         status,
			Diagnostics:    diagnostics,
		},
	)
	if err != nil {
		a.l.Warn("failed to record a report definition's catalog check",
			zap.String("definitionId", def.ID.String()), zap.Error(err))
		return false
	}
	if !recorded || status == def.Status {
		return true
	}

	if status == report.DefinitionStatusNeedsAttention {
		result.NeedsAttention++
	} else {
		result.Restored++
	}
	a.auditCatalogCheck(def, status, version, diagnostics)

	return true
}

func (a *Activities) catalogDiagnostics(def *ir.Definition) []string {
	if def == nil {
		return []string{missingDefinition}
	}
	return reporting.CatalogDiagnostics(a.engine.Validate(def))
}

func catalogCheckStatus(
	current report.DefinitionStatus,
	diagnostics []string,
) report.DefinitionStatus {
	switch {
	case len(diagnostics) > 0 && current == report.DefinitionStatusActive:
		return report.DefinitionStatusNeedsAttention
	case len(diagnostics) == 0 && current == report.DefinitionStatusNeedsAttention:
		return report.DefinitionStatusActive
	default:
		return current
	}
}

func (a *Activities) auditCatalogCheck(
	def *report.ReportDefinition,
	status report.DefinitionStatus,
	version string,
	diagnostics []string,
) {
	if err := a.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     def.ID.String(),
		Operation:      permission.OpUpdate,
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		OrganizationID: def.OrganizationID,
		BusinessUnitID: def.BusinessUnitID,
		PreviousState: map[string]any{
			dataKeyStatus:       string(def.Status),
			dataKeyCatalog:      def.CatalogVersion,
			dataKeyDiagnostics:  def.Diagnostics,
			dataKeyCatalogEvent: catalogCheckEvent,
		},
		CurrentState: map[string]any{
			dataKeyStatus:       string(status),
			dataKeyCatalog:      version,
			dataKeyDiagnostics:  diagnostics,
			dataKeyCatalogEvent: catalogCheckEvent,
		},
	}); err != nil {
		a.l.Warn("failed to audit a report definition's catalog check",
			zap.String("definitionId", def.ID.String()), zap.Error(err))
	}
}
