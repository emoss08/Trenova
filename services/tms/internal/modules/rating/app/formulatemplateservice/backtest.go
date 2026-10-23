package formulatemplateservice

import (
	"context"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	backtestDefaultLimit = 50
	backtestMaxLimit     = 500
	impactDefaultLimit   = 100
	backtestDeadline     = 60 * time.Second
)

type BacktestRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	Source        string
	VersionNumber *int64
	Limit         int
}

type BacktestCurrentSource string

const (
	BacktestCurrentBilled     BacktestCurrentSource = "Billed"
	BacktestCurrentRecomputed BacktestCurrentSource = "Recomputed"
)

type BacktestResult struct {
	ShipmentID       pulid.ID              `json:"shipmentId"`
	ProNumber        string                `json:"proNumber"`
	CurrentSource    BacktestCurrentSource `json:"currentSource"`
	CurrentAmount    decimal.Decimal       `json:"currentAmount"`
	CandidateAmount  decimal.Decimal       `json:"candidateAmount"`
	Delta            decimal.Decimal       `json:"delta"`
	DeltaPct         decimal.Decimal       `json:"deltaPct"`
	CurrentError     string                `json:"currentError,omitempty"`
	CandidateError   string                `json:"candidateError,omitempty"`
	CandidateRefused bool                  `json:"candidateRefused"`
	Adjusted         bool                  `json:"adjusted"`
}

type BacktestSummary = formulatemplateports.BacktestSummary

type BacktestResponse struct {
	Results   []*BacktestResult `json:"results"`
	Summary   BacktestSummary   `json:"summary"`
	Truncated bool              `json:"truncated"`
}

type ratedContent struct {
	source    string
	libraries []rules.TariffLibrary
}

func (s *Service) Backtest(ctx context.Context, req *BacktestRequest) (*BacktestResponse, error) {
	log := s.l.With(
		zap.String("operation", "Backtest"),
		zap.String("templateID", req.TemplateID.String()),
	)

	if err := validateBacktestRequest(req); err != nil {
		return nil, err
	}
	ctx = effectiveversioncache.With(ctx)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	candidate, err := s.backtestCandidate(ctx, req, template)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = backtestDefaultLimit
	}
	return s.rateRecorded(ctx, template, candidate, limit)
}

func validateBacktestRequest(req *BacktestRequest) error {
	multiErr := errortypes.NewMultiError()

	switch {
	case req.Source == "" && req.VersionNumber == nil:
		multiErr.Add(fieldSource, errortypes.ErrRequired, "Either a source or a version number is required")
		multiErr.Add(versionNumberField, errortypes.ErrRequired, "Either a source or a version number is required")
	case req.Source != "" && req.VersionNumber != nil:
		multiErr.Add(fieldSource, errortypes.ErrInvalid, "Provide either a source or a version number, not both")
		multiErr.Add(versionNumberField, errortypes.ErrInvalid, "Provide either a source or a version number, not both")
	case len(req.Source) > tarifftypes.MaxSourceLength:
		multiErr.Add(fieldSource, errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}

	if req.Limit > backtestMaxLimit {
		multiErr.Add("limit", errortypes.ErrInvalid, "Limit cannot exceed 500")
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) backtestCandidate(
	ctx context.Context,
	req *BacktestRequest,
	template *formulatemplate.FormulaTemplate,
) (*ratedContent, error) {
	if req.VersionNumber != nil {
		version, err := s.versionRepo.GetByTemplateAndVersion(ctx, &formulatemplateports.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			TemplateID:    req.TemplateID,
			VersionNumber: *req.VersionNumber,
		})
		if err != nil {
			return nil, err
		}
		return s.pinnedContent(ctx, req.TenantInfo, version.Source, version.Imports)
	}

	return s.latestContent(ctx, templateTenant(template), req.Source)
}

func (s *Service) latestContent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	source string,
) (*ratedContent, error) {
	content, err := s.checkContent(ctx, &contentRequest{
		tenant: tenant,
		source: source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.refusal(); refusal != nil {
		return nil, refusal
	}
	return &ratedContent{source: source, libraries: content.checked.Context.Libraries}, nil
}

func (s *Service) pinnedContent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	source string,
	imports []tarifftypes.Import,
) (*ratedContent, error) {
	libraries, err := s.resolver.PinnedLibraries(ctx, tenant, imports)
	if err != nil {
		return nil, err
	}
	return &ratedContent{source: source, libraries: libraries}, nil
}

type ApprovalImpactRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Limit      int
}

func (s *Service) ApprovalImpact(
	ctx context.Context,
	req *ApprovalImpactRequest,
) (*BacktestResponse, error) {
	if req.Limit > backtestMaxLimit {
		return nil, errortypes.NewValidationError("limit", errortypes.ErrInvalid, "Limit cannot exceed 500")
	}
	ctx = effectiveversioncache.With(ctx)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	candidate, err := s.latestContent(ctx, templateTenant(template), template.Source)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = impactDefaultLimit
	}
	resp, err := s.rateRecorded(ctx, template, candidate, limit)
	if err != nil {
		return nil, err
	}
	sortResultsByImpact(resp.Results)
	return resp, nil
}

func (s *Service) rateRecorded(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	candidate *ratedContent,
	limit int,
) (*BacktestResponse, error) {
	if s.rated == nil {
		return nil, errortypes.NewBusinessError(
			"Backtesting a formula against past shipments is not available on this server",
		)
	}

	tenant := templateTenant(template)
	records, err := s.rated.RatedByTemplate(ctx, &tariffports.RatedByTemplateRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, backtestDeadline)
	defer cancel()

	resp := &BacktestResponse{Results: make([]*BacktestResult, 0, len(records))}
	for _, record := range records {
		if ctx.Err() != nil {
			resp.Truncated = true
			break
		}
		resp.Results = append(resp.Results, s.backtestShipment(ctx, template, candidate, record))
	}
	resp.Summary = buildBacktestSummary(resp.Results)
	return resp, nil
}

func (s *Service) backtestShipment(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	candidate *ratedContent,
	record *tariffports.RatedShipment,
) *BacktestResult {
	result := &BacktestResult{ShipmentID: record.ShipmentID, ProNumber: record.ProNumber}
	tenant := templateTenant(template)

	if record.Billed.Valid {
		result.CurrentSource = BacktestCurrentBilled
		result.CurrentAmount = record.Billed.Decimal
	} else {
		result.CurrentSource = BacktestCurrentRecomputed
		s.recomputeCurrent(ctx, template, record, result)
	}

	rated, err := s.rateSource(ctx, &sourceRating{
		tenant:    tenant,
		name:      template.Name,
		source:    candidate.source,
		libraries: candidate.libraries,
		inputs:    record.Inputs,
		shipment:  record,
	})
	switch {
	case err != nil:
		result.CandidateError = errortypes.LocalizedText(ctx, err)
	case rated.Refusal != nil:
		result.CandidateRefused = true
		result.CandidateError = rated.Refusal.Localize(i18n.FromContext(ctx))
	case rated.Bill != nil:
		result.CandidateAmount = rated.Bill.Total
		result.Adjusted = adjusted(rated.Bill)
	}

	if result.CurrentError == "" && result.CandidateError == "" {
		delta := ratesimulation.Delta{Before: result.CurrentAmount, After: result.CandidateAmount}
		result.Delta = delta.Amount()
		result.DeltaPct = delta.Percent()
	}
	return result
}

func adjusted(bill *tarifftypes.Bill) bool {
	for i := range bill.Lines {
		if bill.Lines[i].Kind == tarifftypes.LineMinimum || bill.Lines[i].Kind == tarifftypes.LineMaximum {
			return true
		}
	}
	return false
}

func (s *Service) recomputeCurrent(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	record *tariffports.RatedShipment,
	result *BacktestResult,
) {
	current, err := s.recordedContent(ctx, template, record)
	if err != nil {
		result.CurrentError = errortypes.LocalizedText(ctx, err)
		return
	}

	rated, err := s.rateSource(ctx, &sourceRating{
		tenant:    templateTenant(template),
		name:      template.Name,
		source:    current.source,
		libraries: current.libraries,
		inputs:    record.Inputs,
		shipment:  record,
	})
	switch {
	case err != nil:
		result.CurrentError = errortypes.LocalizedText(ctx, err)
	case rated.Refusal != nil:
		result.CurrentError = rated.Refusal.Localize(i18n.FromContext(ctx))
	case rated.Bill != nil:
		result.CurrentAmount = rated.Bill.Total
	}
}

func (s *Service) recordedContent(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	record *tariffports.RatedShipment,
) (*ratedContent, error) {
	tenant := templateTenant(template)
	if record.VersionNumber > 0 {
		version, err := effectiveversioncache.GetVersion(
			ctx,
			template.ID,
			record.VersionNumber,
			func(loadCtx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
				return s.lookupVersion(loadCtx, template.ID, record.VersionNumber, tenant)
			},
		)
		if err != nil {
			return nil, err
		}
		if version != nil {
			return s.pinnedContent(ctx, tenant, version.Source, version.Imports)
		}
	}

	asOf := record.RatedAt
	if asOf == 0 {
		asOf = timeutils.NowUnix()
	}
	billing, err := s.resolver.BillingVersion(ctx, template, tenant, asOf)
	if err != nil {
		return nil, err
	}
	content, err := tariff.BillingContent(template, billing)
	if err != nil {
		return nil, err
	}
	return s.pinnedContent(ctx, tenant, content.Source, content.Imports)
}

func (s *Service) lookupVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionNumber int64,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	version, err := s.versionRepo.GetByTemplateAndVersion(ctx, &formulatemplateports.GetVersionRequest{
		TenantInfo:    tenantInfo,
		TemplateID:    templateID,
		VersionNumber: versionNumber,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) || dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a stamp that points at nothing falls back to billing
		}
		return nil, err
	}
	return version, nil
}

func buildBacktestSummary(results []*BacktestResult) BacktestSummary {
	var acc ratesimulation.Accumulator
	var currentErrors, candidateErrors, refused, adjustedCount int

	for _, result := range results {
		failed := result.CurrentError != "" || result.CandidateError != ""
		acc.Add(ratesimulation.Delta{
			Before: result.CurrentAmount,
			After:  result.CandidateAmount,
			Failed: failed,
		})
		if result.CurrentError != "" {
			currentErrors++
		}
		if result.CandidateError != "" {
			candidateErrors++
		}
		if result.CandidateRefused {
			refused++
		}
		if result.Adjusted && !failed {
			adjustedCount++
		}
	}

	shared := acc.Summary()
	return BacktestSummary{
		ShipmentCount:       shared.ShipmentCount,
		EvaluatedCount:      shared.EvaluatedCount,
		ChangedCount:        shared.ChangedCount,
		IncreasedCount:      shared.IncreasedCount,
		DecreasedCount:      shared.DecreasedCount,
		ErrorCount:          shared.ErrorCount,
		CurrentErrorCount:   currentErrors,
		CandidateErrorCount: candidateErrors,
		RefusedCount:        refused,
		AdjustedCount:       adjustedCount,
		CurrentTotal:        shared.BeforeTotal,
		CandidateTotal:      shared.AfterTotal,
		TotalDelta:          shared.TotalDelta,
		TotalDeltaPct:       shared.TotalDeltaPct,
		MaxIncrease:         shared.MaxIncrease,
		MaxDecrease:         shared.MaxDecrease,
	}
}

func sortResultsByImpact(results []*BacktestResult) {
	sort.SliceStable(results, func(i, j int) bool {
		iFailed := results[i].CurrentError != "" || results[i].CandidateError != ""
		jFailed := results[j].CurrentError != "" || results[j].CandidateError != ""
		if iFailed != jFailed {
			return jFailed
		}
		return results[i].Delta.Abs().GreaterThan(results[j].Delta.Abs())
	})
}
