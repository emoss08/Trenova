package formulatemplatereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type ReviewDiffRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

type ReviewDiffResponse struct {
	HasApprovedBase  bool                             `json:"hasApprovedBase"`
	BaseVersion      int64                            `json:"baseVersion"`
	CurrentVersion   int64                            `json:"currentVersion"`
	BaseSource       string                           `json:"baseSource"`
	CurrentSource    string                           `json:"currentSource"`
	BaseImports      []tarifftypes.Import             `json:"baseImports"`
	CurrentImports   []tarifftypes.Import             `json:"currentImports"`
	Changes          map[string]jsonutils.FieldChange `json:"changes"`
	ChangeCount      int                              `json:"changeCount"`
	PendingSchedules []ScheduledWindow                `json:"pendingSchedules"`
}

type ScheduledWindow struct {
	VersionNumber int64 `json:"versionNumber"`
	EffectiveFrom int64 `json:"effectiveFrom"`
}

func (s *Service) ReviewDiff(
	ctx context.Context,
	req *ReviewDiffRequest,
) (*ReviewDiffResponse, error) {
	log := s.l.With(
		zap.String("operation", "ReviewDiff"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.templates.GetTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	approved, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulatemplateports.GetLatestVersionByStatusRequest{
			TenantInfo: req.TenantInfo,
			TemplateID: req.TemplateID,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil {
		log.Error("failed to get last approved snapshot", zap.Error(err))
		return nil, err
	}

	current := formulatemplate.NewVersionFromTemplate(
		template,
		template.CurrentVersionNumber,
		req.TenantInfo.UserID,
		"",
		nil,
	)

	resp := &ReviewDiffResponse{
		CurrentVersion: template.CurrentVersionNumber,
		CurrentSource:  template.Source,
		CurrentImports: nonNilImports(template.Imports),
		BaseImports:    []tarifftypes.Import{},
	}

	base := &formulatemplate.FormulaTemplateVersion{}
	if approved != nil {
		base = approved
		resp.HasApprovedBase = true
		resp.BaseVersion = approved.VersionNumber
		resp.BaseSource = approved.Source
		resp.BaseImports = nonNilImports(approved.Imports)
	}

	changes, err := jsonutils.JSONDiff(base, current, &jsonutils.DiffOptions{
		IgnoreFields: formulatemplateservice.VersionDiffIgnoreFields,
	})
	if err != nil {
		log.Error("failed to compute review diff", zap.Error(err))
		return nil, err
	}

	resp.Changes = changes
	resp.ChangeCount = len(changes)

	resp.PendingSchedules, err = s.pendingSchedules(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to list pending schedules", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func (s *Service) pendingSchedules(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]ScheduledWindow, error) {
	scheduled, err := s.versionRepo.ListScheduled(
		ctx,
		&formulatemplateports.ListScheduledVersionsRequest{
			TenantInfo: tenantInfo,
			TemplateID: templateID,
		},
	)
	if err != nil {
		return nil, err
	}

	windows := make([]ScheduledWindow, 0, len(scheduled))
	for _, version := range scheduled {
		if version == nil || version.EffectiveFrom == nil {
			continue
		}
		windows = append(windows, ScheduledWindow{
			VersionNumber: version.VersionNumber,
			EffectiveFrom: *version.EffectiveFrom,
		})
	}

	return windows, nil
}

func nonNilImports(imports []tarifftypes.Import) []tarifftypes.Import {
	if imports == nil {
		return []tarifftypes.Import{}
	}
	return imports
}
