package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratesimulation"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	backtestDefaultLimit = 50
	backtestMaxLimit     = 500
)

type BacktestRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	Expression    string
	VersionNumber *int64
	Limit         int
}

type BacktestResult struct {
	ShipmentID       pulid.ID        `json:"shipmentId"`
	ProNumber        string          `json:"proNumber"`
	CurrentAmount    decimal.Decimal `json:"currentAmount"`
	CandidateAmount  decimal.Decimal `json:"candidateAmount"`
	Delta            decimal.Decimal `json:"delta"`
	DeltaPct         decimal.Decimal `json:"deltaPct"`
	CurrentError     string          `json:"currentError,omitempty"`
	CandidateError   string          `json:"candidateError,omitempty"`
	GuardrailApplied bool            `json:"guardrailApplied"`
}

// BacktestSummary is what a whole backtest came to.
//
// The shape is ratesimulation.Summary with two fields renamed, because a
// backtest calls them current and candidate while a rate simulation calls them
// before and after. The arithmetic is shared; only the words differ.
type BacktestSummary struct {
	ShipmentCount  int             `json:"shipmentCount"`
	EvaluatedCount int             `json:"evaluatedCount"`
	ChangedCount   int             `json:"changedCount"`
	IncreasedCount int             `json:"increasedCount"`
	DecreasedCount int             `json:"decreasedCount"`
	ErrorCount     int             `json:"errorCount"`
	CurrentTotal   decimal.Decimal `json:"currentTotal"`
	CandidateTotal decimal.Decimal `json:"candidateTotal"`
	TotalDelta     decimal.Decimal `json:"totalDelta"`
	TotalDeltaPct  decimal.Decimal `json:"totalDeltaPct"`
	MaxIncrease    decimal.Decimal `json:"maxIncrease"`
	MaxDecrease    decimal.Decimal `json:"maxDecrease"`
}

type BacktestResponse struct {
	Results []*BacktestResult `json:"results"`
	Summary BacktestSummary   `json:"summary"`
}

func (s *Service) Backtest(
	ctx context.Context,
	req *BacktestRequest,
) (*BacktestResponse, error) {
	log := s.l.With(
		zap.String("operation", "Backtest"),
		zap.String("templateID", req.TemplateID.String()),
	)

	if err := validateBacktestRequest(req); err != nil {
		return nil, err
	}

	ctx = ratetablecache.With(ctx)
	ctx = effectiveversioncache.With(ctx)

	limit := req.Limit
	if limit <= 0 {
		limit = backtestDefaultLimit
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	candidate, err := s.resolveBacktestCandidate(ctx, req, template)
	if err != nil {
		log.Error("failed to resolve backtest candidate", zap.Error(err))
		return nil, err
	}

	shipments, err := s.shipmentRepo.ListRatedByFormulaTemplate(
		ctx,
		&repositories.ListRatedByFormulaTemplateRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			Limit:      limit,
		},
	)
	if err != nil {
		log.Error("failed to list rated shipments", zap.Error(err))
		return nil, err
	}

	results := make([]*BacktestResult, 0, len(shipments))
	for _, entity := range shipments {
		results = append(
			results,
			s.backtestShipment(ctx, template, candidate, entity, req.TenantInfo),
		)
	}

	return &BacktestResponse{
		Results: results,
		Summary: buildBacktestSummary(results),
	}, nil
}

func validateBacktestRequest(req *BacktestRequest) error {
	multiErr := errortypes.NewMultiError()

	switch {
	case req.Expression == "" && req.VersionNumber == nil:
		multiErr.Add(
			"expression",
			errortypes.ErrRequired,
			"Either an expression or a version number is required",
		)
		multiErr.Add(
			"versionNumber",
			errortypes.ErrRequired,
			"Either an expression or a version number is required",
		)
	case req.Expression != "" && req.VersionNumber != nil:
		multiErr.Add(
			"expression",
			errortypes.ErrInvalid,
			"Provide either an expression or a version number, not both",
		)
		multiErr.Add(
			"versionNumber",
			errortypes.ErrInvalid,
			"Provide either an expression or a version number, not both",
		)
	}

	if req.Limit > backtestMaxLimit {
		multiErr.Add(
			"limit",
			errortypes.ErrInvalid,
			"Limit cannot exceed 500",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) resolveBacktestCandidate(
	ctx context.Context,
	req *BacktestRequest,
	template *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	if req.VersionNumber != nil {
		version, err := s.versionRepo.GetByTemplateAndVersion(ctx, &repositories.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			TemplateID:    req.TemplateID,
			VersionNumber: *req.VersionNumber,
		})
		if err != nil {
			return nil, err
		}

		return template.ApplyVersion(version), nil
	}

	candidate := *template
	candidate.Expression = req.Expression

	return &candidate, nil
}

func (s *Service) backtestShipment(
	ctx context.Context,
	template, candidate *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) *BacktestResult {
	result := &BacktestResult{
		ShipmentID: entity.ID,
		ProNumber:  entity.ProNumber,
	}

	current, err := s.resolveEffectiveForShipment(ctx, template, entity, tenantInfo)
	if err != nil {
		result.CurrentError = err.Error()
	} else if resp, rateErr := s.formulaService.Rate(ctx, &formula.RateRequest{
		Template: current,
		Entity:   entity,
	}); rateErr != nil {
		result.CurrentError = rateErr.Error()
	} else {
		result.CurrentAmount = resp.Amount
		if resp.Guardrail != nil && resp.Guardrail.Applied {
			result.GuardrailApplied = true
		}
	}

	if resp, rateErr := s.formulaService.Rate(ctx, &formula.RateRequest{
		Template: candidate,
		Entity:   entity,
	}); rateErr != nil {
		result.CandidateError = rateErr.Error()
	} else {
		result.CandidateAmount = resp.Amount
		if resp.Guardrail != nil && resp.Guardrail.Applied {
			result.GuardrailApplied = true
		}
	}

	if result.CurrentError == "" && result.CandidateError == "" {
		delta := ratesimulation.Delta{
			Before: result.CurrentAmount,
			After:  result.CandidateAmount,
		}

		result.Delta = delta.Amount()
		result.DeltaPct = delta.Percent()
	}

	return result
}

func (s *Service) resolveEffectiveForShipment(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplate, error) {
	asOf := entity.CreatedAt
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		asOf = *entity.ActualShipDate
	}
	if asOf == 0 {
		asOf = timeutils.NowUnix()
	}

	return s.formulaService.ResolveEffectiveTemplate(ctx, template, tenantInfo, asOf)
}

func buildBacktestSummary(results []*BacktestResult) BacktestSummary {
	var acc ratesimulation.Accumulator

	for _, result := range results {
		acc.Add(ratesimulation.Delta{
			Before: result.CurrentAmount,
			After:  result.CandidateAmount,
			Failed: result.CurrentError != "" || result.CandidateError != "",
		})
	}

	shared := acc.Summary()

	return BacktestSummary{
		ShipmentCount:  shared.ShipmentCount,
		EvaluatedCount: shared.EvaluatedCount,
		ChangedCount:   shared.ChangedCount,
		IncreasedCount: shared.IncreasedCount,
		DecreasedCount: shared.DecreasedCount,
		ErrorCount:     shared.ErrorCount,
		CurrentTotal:   shared.BeforeTotal,
		CandidateTotal: shared.AfterTotal,
		TotalDelta:     shared.TotalDelta,
		TotalDeltaPct:  shared.TotalDeltaPct,
		MaxIncrease:    shared.MaxIncrease,
		MaxDecrease:    shared.MaxDecrease,
	}
}
