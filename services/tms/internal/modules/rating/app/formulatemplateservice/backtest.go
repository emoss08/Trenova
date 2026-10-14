package formulatemplateservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/dberror"
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
	GuardrailApplied bool                  `json:"guardrailApplied"`
}

type BacktestSummary = formulatemplateports.BacktestSummary

type BacktestResponse struct {
	Results   []*BacktestResult `json:"results"`
	Summary   BacktestSummary   `json:"summary"`
	Truncated bool              `json:"truncated"`
}

const backtestDeadline = 60 * time.Second

type rateEachRequest struct {
	current        *formulatemplate.FormulaTemplate
	candidate      *formulatemplate.FormulaTemplate
	shipments      []*shipment.Shipment
	tenantInfo     pagination.TenantInfo
	billedBaseline bool
}

func (s *Service) rateEach(ctx context.Context, req *rateEachRequest) *BacktestResponse {
	ctx, cancel := context.WithTimeout(
		engine.WithEvaluationTimeout(ctx, previewEvaluationTimeout),
		backtestDeadline,
	)
	defer cancel()

	resp := &BacktestResponse{Results: make([]*BacktestResult, 0, len(req.shipments))}
	for _, entity := range req.shipments {
		if ctx.Err() != nil {
			resp.Truncated = true
			break
		}
		resp.Results = append(resp.Results, s.backtestShipment(ctx, req, entity))
	}
	resp.Summary = buildBacktestSummary(resp.Results)

	return resp
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
	ctx = contextvariablecache.With(ctx)
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

	shipments, err := s.listRatedShipments(ctx, req.TemplateID, req.TenantInfo, limit)
	if err != nil {
		log.Error("failed to list rated shipments", zap.Error(err))
		return nil, err
	}

	return s.rateEach(ctx, &rateEachRequest{
		current:        template,
		candidate:      candidate,
		shipments:      shipments,
		tenantInfo:     req.TenantInfo,
		billedBaseline: true,
	}), nil
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
		version, err := s.versionRepo.GetByTemplateAndVersion(
			ctx,
			&formulatemplateports.GetVersionRequest{
				TenantInfo:    req.TenantInfo,
				TemplateID:    req.TemplateID,
				VersionNumber: *req.VersionNumber,
			},
		)
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
	req *rateEachRequest,
	entity *shipment.Shipment,
) *BacktestResult {
	result := &BacktestResult{
		ShipmentID: entity.ID,
		ProNumber:  entity.ProNumber,
	}

	if billed, ok := billedFormulaAmount(req.current, entity); req.billedBaseline && ok {
		result.CurrentSource = BacktestCurrentBilled
		result.CurrentAmount = billed
		result.GuardrailApplied = entity.RatingDetail.Guardrail != nil &&
			entity.RatingDetail.Guardrail.Applied
	} else {
		result.CurrentSource = BacktestCurrentRecomputed
		s.recomputeCurrent(ctx, req, entity, result)
	}

	if resp, rateErr := s.formulaService.Rate(ctx, &formula.RateRequest{
		Template: req.candidate,
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

func (s *Service) recomputeCurrent(
	ctx context.Context,
	req *rateEachRequest,
	entity *shipment.Shipment,
	result *BacktestResult,
) {
	current, err := s.resolveCurrentForShipment(ctx, req.current, entity, req.tenantInfo)
	if err != nil {
		result.CurrentError = err.Error()
		return
	}

	resp, err := s.formulaService.Rate(ctx, &formula.RateRequest{
		Template: current,
		Entity:   entity,
	})
	if err != nil {
		result.CurrentError = err.Error()
		return
	}

	result.CurrentAmount = resp.Amount
	if resp.Guardrail != nil && resp.Guardrail.Applied {
		result.GuardrailApplied = true
	}
}

func billedFormulaAmount(
	template *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
) (decimal.Decimal, bool) {
	detail := entity.RatingDetail
	if detail == nil || detail.FormulaTemplateID != template.ID.String() {
		return decimal.Zero, false
	}
	if !entity.FreightChargeAmount.Valid || entity.RateOverrideAmount.Valid {
		return decimal.Zero, false
	}

	return entity.FreightChargeAmount.Decimal, true
}

// resolveCurrentForShipment reconstructs the content a shipment was actually
// charged with, for the rows whose "current" side has to be recomputed: an
// approval impact, which isolates the content change from rate-table drift, and
// a backtest row with no stored formula charge to compare against.
//
// The row is the wrong baseline more often than not: while a template is in
// review the row already carries the pending edit, and a template under
// revision has been demoted to Draft. So the search goes from most to least
// specific — the version stamped on the shipment's own rating detail, then the
// approved snapshot in force on its rating date — and only falls back to the
// row when the template has no approved history at all.
func (s *Service) resolveCurrentForShipment(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplate, error) {
	if recorded := recordedVersionNumber(template, entity); recorded > 0 {
		version, err := effectiveversioncache.GetVersion(
			ctx,
			template.ID,
			recorded,
			func(loadCtx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
				return s.lookupVersion(loadCtx, template.ID, recorded, tenantInfo)
			},
		)
		if err != nil {
			return nil, err
		}
		if version != nil {
			return template.ApplyVersion(version), nil
		}
	}

	billing, err := s.formulaService.ResolveBillingVersion(
		ctx,
		template,
		tenantInfo,
		entity.RatingAsOf(timeutils.NowUnix()),
	)
	if err != nil {
		return nil, err
	}
	if billing != nil {
		return template.ApplyVersion(billing), nil
	}

	return template, nil
}

// lookupVersion fetches one numbered snapshot, treating "no such version" as
// nil so a stale stamp on a shipment degrades to the next resolution step
// instead of failing the whole backtest row.
func (s *Service) lookupVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionNumber int64,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	version, err := s.versionRepo.GetByTemplateAndVersion(
		ctx,
		&formulatemplateports.GetVersionRequest{
			TenantInfo:    tenantInfo,
			TemplateID:    templateID,
			VersionNumber: versionNumber,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) || dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil version means the stamp points at nothing
		}
		return nil, err
	}

	return version, nil
}

func recordedVersionNumber(
	template *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
) int64 {
	detail := entity.RatingDetail
	if detail == nil || detail.FormulaTemplateID != template.ID.String() {
		return 0
	}

	return detail.VersionNumber
}

func buildBacktestSummary(results []*BacktestResult) BacktestSummary {
	var acc ratesimulation.Accumulator
	var currentErrors, candidateErrors, clamps int

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
		if result.GuardrailApplied && !failed {
			clamps++
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
		GuardrailCount:      clamps,
		CurrentTotal:        shared.BeforeTotal,
		CandidateTotal:      shared.AfterTotal,
		TotalDelta:          shared.TotalDelta,
		TotalDeltaPct:       shared.TotalDeltaPct,
		MaxIncrease:         shared.MaxIncrease,
		MaxDecrease:         shared.MaxDecrease,
	}
}

func (s *Service) listRatedShipments(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*shipment.Shipment, error) {
	shipments, err := s.shipmentRepo.ListRatedByFormulaTemplate(
		ctx,
		&shipmentports.ListRatedByFormulaTemplateRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
			Limit:      limit,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = s.hydrateShipments(ctx, shipments, tenantInfo); err != nil {
		return nil, err
	}

	return shipments, nil
}

func (s *Service) hydrateShipments(
	ctx context.Context,
	shipments []*shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) error {
	if s.hydrator == nil {
		return nil
	}

	return s.hydrator.HydrateShipments(ctx, &ratequoteports.HydrateShipmentsRequest{
		Shipments:  shipments,
		TenantInfo: tenantInfo,
	})
}
