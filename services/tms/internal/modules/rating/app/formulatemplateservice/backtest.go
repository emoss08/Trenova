package formulatemplateservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
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
	unresolved     map[pulid.ID]error
}

func (s *Service) rateEach(ctx context.Context, req *rateEachRequest) *BacktestResponse {
	ctx, cancel := context.WithTimeout(ctx, backtestDeadline)
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

	shipments, unresolved, err := s.listRatedShipments(ctx, req.TemplateID, req.TenantInfo, limit)
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
		unresolved:     unresolved,
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

	unresolved := req.unresolved[entity.ID]
	switch billed, ok := billedFormulaAmount(req.current, entity); {
	case req.billedBaseline && ok:
		result.CurrentSource = BacktestCurrentBilled
		result.CurrentAmount = billed
		result.GuardrailApplied = entity.RatingDetail.Guardrail != nil &&
			entity.RatingDetail.Guardrail.Applied
	case unresolved != nil:
		result.CurrentSource = BacktestCurrentRecomputed
		result.CurrentError = errortypes.LocalizedText(ctx, unresolved)
	default:
		result.CurrentSource = BacktestCurrentRecomputed
		s.recomputeCurrent(ctx, req, entity, result)
	}

	if unresolved != nil {
		result.CandidateError = errortypes.LocalizedText(ctx, unresolved)
	} else if resp, rateErr := s.rateLeashed(ctx, req.candidate, entity); rateErr != nil {
		result.CandidateError = errortypes.LocalizedText(ctx, rateErr)
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
		result.CurrentError = errortypes.LocalizedText(ctx, err)
		return
	}

	resp, err := s.rateLeashed(ctx, current, entity)
	if err != nil {
		result.CurrentError = errortypes.LocalizedText(ctx, err)
		return
	}

	result.CurrentAmount = resp.Amount
	if resp.Guardrail != nil && resp.Guardrail.Applied {
		result.GuardrailApplied = true
	}
}

func (s *Service) rateLeashed(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	entity *shipment.Shipment,
) (*formulatemplatetypes.CalculateResponse, error) {
	leashed, cancel := context.WithTimeout(ctx, previewEvaluationTimeout)
	defer cancel()

	resp, err := s.formulaService.Rate(leashed, &formula.RateRequest{
		Template: template,
		Entity:   entity,
	})
	if err != nil && ctx.Err() == nil && errors.Is(leashed.Err(), context.DeadlineExceeded) {
		return nil, timeoutMessage().ValidationError("expression", errortypes.ErrInvalid)
	}
	return resp, err
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
// charged with, so the "current" side of a backtest is the number on the
// invoice rather than whatever the template row happens to hold today.
//
// The row is the wrong baseline more often than not: while a template is in
// review the row already carries the pending edit, and a template under
// revision has been demoted to Draft. So the search goes from most to least
// specific — the version stamped on the shipment's own rating detail, then a
// version scheduled into effect for its ship date, then the newest snapshot
// the template took while Active — and only falls back to the row when the
// template has no approved history at all.
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
	if billing.InForce != nil {
		return template.ApplyVersion(billing.InForce), nil
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
) ([]*shipment.Shipment, map[pulid.ID]error, error) {
	shipments, err := s.shipmentRepo.ListRatedByFormulaTemplate(
		ctx,
		&shipmentports.ListRatedByFormulaTemplateRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfo,
			Limit:      limit,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	unresolved, err := s.hydrateShipments(ctx, shipments, tenantInfo)
	if err != nil {
		return nil, nil, err
	}

	return shipments, unresolved, nil
}

func (s *Service) hydrateShipments(
	ctx context.Context,
	shipments []*shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]error, error) {
	if s.hydrator == nil {
		return nil, nil
	}

	err := s.hydrator.HydrateShipments(ctx, &ratequoteports.HydrateShipmentsRequest{
		Shipments:  shipments,
		TenantInfo: tenantInfo,
	})
	if err == nil {
		return nil, nil
	}
	if !unresolvedReference(err) {
		return nil, err
	}

	unresolved := make(map[pulid.ID]error)
	for _, entity := range shipments {
		if entity == nil {
			continue
		}
		hydrateErr := s.hydrateShipment(ctx, entity, tenantInfo)
		switch {
		case hydrateErr == nil:
		case unresolvedReference(hydrateErr):
			unresolved[entity.ID] = hydrateErr
		default:
			return nil, hydrateErr
		}
	}

	return unresolved, nil
}

func (s *Service) hydrateShipment(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) error {
	if s.hydrator == nil {
		return nil
	}

	return s.hydrator.HydrateShipment(ctx, &ratequoteports.HydrateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo,
	})
}

func unresolvedReference(err error) bool {
	var validation *errortypes.Error
	return errors.As(err, &validation) && validation.Code == errortypes.ErrInvalidReference
}
