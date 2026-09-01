package formula

import (
	"context"
	goErrors "errors"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	formulaerrors "github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger         *zap.Logger
	Registry       *schema.Registry
	Engine         *engine.Engine
	Resolver       *resolver.Resolver
	Repo           repositories.FormulaTemplateRepository
	VersionRepo    repositories.FormulaTemplateVersionRepository
	RateMatrixRepo repositories.RateMatrixRepository
}

type Service struct {
	l              *zap.Logger
	registry       *schema.Registry
	engine         *engine.Engine
	resolver       *resolver.Resolver
	repo           repositories.FormulaTemplateRepository
	versionRepo    repositories.FormulaTemplateVersionRepository
	rateMatrixRepo repositories.RateMatrixRepository
}

//nolint:gocritic // fx param structs are passed by value
func NewService(p ServiceParams) *Service {
	resolver.RegisterDefaultComputed(p.Resolver)

	return &Service{
		l:              p.Logger.Named("service.formula"),
		registry:       p.Registry,
		engine:         p.Engine,
		resolver:       p.Resolver,
		repo:           p.Repo,
		versionRepo:    p.VersionRepo,
		rateMatrixRepo: p.RateMatrixRepo,
	}
}

func (s *Service) Calculate(
	ctx context.Context,
	req *formulatemplatetypes.CalculateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "Calculate"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.repo.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	if template.Status != formulatemplate.StatusActive {
		return nil, errortypes.NewValidationError(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"Formula template must be Active to rate shipments",
		)
	}

	resolved, err := s.ResolveEffectiveTemplate(ctx, template, req.TenantInfo, req.RatingDate)
	if err != nil {
		log.Error("failed to resolve effective template version", zap.Error(err))
		return nil, err
	}

	resp, err := s.Rate(ctx, &RateRequest{
		Template:  resolved,
		Entity:    req.Entity,
		Variables: req.Variables,
		Overrides: req.Overrides,
	})
	if err != nil {
		log.Error("failed to evaluate formula", zap.Error(err))
		return nil, err
	}

	if resp.Guardrail != nil && resp.Guardrail.Applied {
		log.Warn("formula guardrail applied",
			zap.String("bound", resp.Guardrail.Bound),
			zap.String("rawAmount", resp.Guardrail.RawAmount.String()),
			zap.String("clampedAmount", resp.Amount.String()),
		)
	}

	return resp, nil
}

// ResolveEffectiveTemplate resolves the template content that is in effect for
// the given rating date, honouring scheduled version activations. The version
// list is memoized per unit of work via effectiveversioncache when a caller
// installed one, so batch re-rating does not query per shipment.
func (s *Service) ResolveEffectiveTemplate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	ratingDate int64,
) (*formulatemplate.FormulaTemplate, error) {
	version, err := s.ResolveScheduledVersion(ctx, template, tenantInfo, ratingDate)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return template, nil
	}

	return template.ApplyVersion(version), nil
}

// ResolveScheduledVersion returns the scheduled snapshot in effect for the
// rating date, or nil when none is scheduled that early. Callers that need to
// know whether a schedule applied at all, rather than just what content to
// use, ask this directly.
func (s *Service) ResolveScheduledVersion(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	ratingDate int64,
) (*formulatemplate.FormulaTemplateVersion, error) {
	asOf := ratingDate
	if asOf == 0 {
		asOf = timeutils.NowUnix()
	}

	versions, err := effectiveversioncache.GetVersions(
		ctx,
		template.ID,
		func(loadCtx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
			return s.versionRepo.ListScheduled(
				loadCtx,
				&repositories.ListScheduledVersionsRequest{
					TenantInfo: tenantInfo,
					TemplateID: template.ID,
				},
			)
		},
	)
	if err != nil {
		return nil, err
	}

	return effectiveversioncache.EffectiveAsOf(versions, asOf), nil
}

type RateRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
	Overrides map[string]any
}

func (s *Service) Rate(
	ctx context.Context,
	req *RateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	lookup, err := s.BuildLookup(ctx, pagination.TenantInfo{
		OrgID: req.Template.OrganizationID,
		BuID:  req.Template.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	result, err := s.engine.Evaluate(ctx, &formulatemplatetypes.EvaluationRequest{
		Template:  req.Template,
		Entity:    req.Entity,
		Variables: req.Variables,
		Overrides: req.Overrides,
		Lookup:    lookup,
	})
	if err != nil {
		// A record missing a field the formula never guarded is an authoring
		// problem with a clear fix, and the caller should see it as one rather
		// than as an internal failure.
		var missing *formulaerrors.MissingFieldError
		if goErrors.As(err, &missing) {
			return nil, errortypes.NewValidationError(
				"expression",
				errortypes.ErrInvalid,
				missing.Error(),
			)
		}
		return nil, err
	}

	amount, guardrail, rounding := ApplyChargePolicy(req.Template.ChargePolicy(), result.Value)

	return &formulatemplatetypes.CalculateResponse{
		Amount:              amount,
		Variables:           result.Variables,
		FormulaTemplateID:   req.Template.ID.String(),
		FormulaTemplateName: req.Template.Name,
		Expression:          req.Template.Expression,
		Breakdown:           result.Breakdown,
		Guardrail:           guardrail,
		Rounding:            rounding,
		VersionNumber:       req.Template.CurrentVersionNumber,
	}, nil
}

// BuildLookup reads the tenant's lookup tables, once per unit of work.
//
// A lookup table is a single-axis rate matrix: the expression language calls
// them tables and the storage calls them matrices, and this is where the two
// vocabularies meet. Reading them is expensive — every one-axis matrix with
// every cell — and rating now happens on every shipment write and in batches.
// The memo is only present when a caller installed one; without it this
// behaves exactly as it always did, which is what keeps a formula evaluated
// outside a request working.
func (s *Service) BuildLookup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (formulatemplatetypes.RateTableLookup, error) {
	return ratetablecache.Get(
		ctx,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		func(ctx context.Context) (formulatemplatetypes.RateTableLookup, error) {
			data, err := s.rateMatrixRepo.GetLookupData(
				ctx,
				&repositories.GetRateMatrixLookupDataRequest{TenantInfo: tenantInfo},
			)
			if err != nil {
				return nil, err
			}

			return NewMatrixLookup(data), nil
		},
	)
}

// ApplyChargePolicy turns a raw evaluation into the billable amount: clamp to
// the guardrails, then round. Guardrails go first so a floor of $250.00 is
// the exact floor, not $250.00 rounded to whatever the policy says. Every
// surface that shows an amount — production rating, the Studio preview, a
// saved scenario, a backtest — comes through here, which is what makes the
// number on the screen the number on the invoice.
func ApplyChargePolicy(
	policy formulatypes.ChargePolicy,
	rawAmount decimal.Decimal,
) (decimal.Decimal, *formulatemplatetypes.GuardrailResult, *formulatemplatetypes.RoundingResult) {
	policy = policy.Normalized()

	clamped, guardrail := ApplyGuardrailBounds(policy.MinCharge, policy.MaxCharge, rawAmount)
	rounded := policy.RoundingMode.Round(clamped, policy.RoundingPrecision)

	rounding := &formulatemplatetypes.RoundingResult{
		Mode:            policy.RoundingMode.String(),
		Precision:       policy.RoundingPrecision,
		Applied:         !rounded.Equal(clamped),
		UnroundedAmount: clamped,
	}

	return rounded, guardrail, rounding
}

func ApplyGuardrailBounds(
	minCharge, maxCharge decimal.NullDecimal,
	rawAmount decimal.Decimal,
) (decimal.Decimal, *formulatemplatetypes.GuardrailResult) {
	if !minCharge.Valid && !maxCharge.Valid {
		return rawAmount, nil
	}

	guardrail := &formulatemplatetypes.GuardrailResult{RawAmount: rawAmount}
	if minCharge.Valid {
		lowerBound := minCharge.Decimal
		guardrail.MinCharge = &lowerBound
	}
	if maxCharge.Valid {
		upperBound := maxCharge.Decimal
		guardrail.MaxCharge = &upperBound
	}

	amount := rawAmount
	switch {
	case minCharge.Valid && rawAmount.LessThan(minCharge.Decimal):
		amount = minCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "min"
	case maxCharge.Valid && rawAmount.GreaterThan(maxCharge.Decimal):
		amount = maxCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "max"
	}

	return amount, guardrail
}

type EvaluateExpressionRequest struct {
	Expression string
	Entity     any
	SchemaID   string
	Variables  map[string]any
	Breakdowns []*formulatypes.BreakdownDefinition
	TenantInfo pagination.TenantInfo
}

func (s *Service) EvaluateExpression(
	ctx context.Context,
	req *EvaluateExpressionRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "EvaluateExpression"),
		zap.String("schemaID", req.SchemaID),
	)

	var lookup formulatemplatetypes.RateTableLookup
	if !req.TenantInfo.OrgID.IsNil() {
		builtLookup, err := s.BuildLookup(ctx, req.TenantInfo)
		if err != nil {
			log.Error("failed to build rate table lookup", zap.Error(err))
			return nil, err
		}
		lookup = builtLookup
	}

	result, err := s.engine.EvaluateExpression(
		ctx,
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: req.Expression,
			Entity:     req.Entity,
			SchemaID:   req.SchemaID,
			Variables:  req.Variables,
			Breakdowns: req.Breakdowns,
			Lookup:     lookup,
		},
	)
	if err != nil {
		log.Error("failed to evaluate expression", zap.Error(err))
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
		Breakdown: result.Breakdown,
	}, nil
}

// EvaluatePredicate answers a yes-or-no question about an entity.
//
// The expression language has no boolean result type of its own — every
// evaluation lands on a decimal, with true and false arriving as one and zero —
// so the truth test is "not zero". That also makes a numeric condition like
// `totalStops - 2` behave the way somebody writing it would expect.
//
// No rate-table lookup is built. A condition that reaches for a rate table is
// asking the wrong question, and building one would put a full tenant table
// load on the save path of every shipment. The stub is named here on purpose:
// a table reference in a predicate reads as zero, which is documented, rather
// than as an error that would block the save.
func (s *Service) EvaluatePredicate(
	ctx context.Context,
	req *services.EvaluatePredicateRequest,
) (bool, error) {
	result, err := s.engine.EvaluateExpression(
		ctx,
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression:   req.Expression,
			Entity:       req.Entity,
			SchemaID:     req.SchemaID,
			Lookup:       engine.StubLookup{},
			AllowBoolean: true,
		},
	)
	if err != nil {
		s.l.Warn("failed to evaluate predicate",
			zap.String("operation", "EvaluatePredicate"),
			zap.String("schemaId", req.SchemaID),
			zap.Error(err),
		)

		return false, err
	}

	return !result.Value.IsZero(), nil
}

func (s *Service) ValidateExpression(ctx context.Context, expression, schemaID string) error {
	return s.engine.ValidateExpression(ctx, expression, schemaID)
}

func (s *Service) EvaluateWithEnv(
	ctx context.Context,
	req *formulatemplatetypes.EnvEvaluationRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	result, err := s.engine.EvaluateWithEnv(ctx, req)
	if err != nil {
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
	}, nil
}

func (s *Service) ValidateExpressionWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) error {
	return s.engine.ValidateExpressionWithEnv(ctx, expression, env)
}

// UnguardedNullableFields lists the nullable schema fields an expression uses
// without a guard, so an author hears about it while typing rather than from
// the first shipment that fails to rate.
func (s *Service) UnguardedNullableFields(
	ctx context.Context,
	expression string,
	schemaID string,
	variables map[string]any,
) ([]engine.NullableFieldWarning, error) {
	return s.engine.UnguardedNullableFields(ctx, expression, schemaID, variables)
}

func (s *Service) ValidateExpressionDetailed(
	ctx context.Context,
	expression string,
	env map[string]any,
) engine.ValidationOutcome {
	return s.engine.ValidateExpressionDetailed(ctx, expression, env)
}

func (s *Service) ValidateLookupTables(
	ctx context.Context,
	expression string,
	tenantInfo pagination.TenantInfo,
) error {
	refs, err := engine.ExtractLookupTableRefs(expression)
	if err != nil || (len(refs.Single) == 0 && len(refs.Multi) == 0) {
		return nil //nolint:nilerr // unparseable expressions are rejected by compile validation
	}

	lookup, err := s.BuildLookup(ctx, tenantInfo)
	if err != nil {
		return err
	}

	multiErr := errortypes.NewMultiError()
	for _, table := range refs.Single {
		if lookup.Has(table) {
			continue
		}
		if lookup.Has2(table) {
			multiErr.Add(
				"expression",
				errortypes.ErrInvalid,
				"Rate table "+table+
					" has two axes — address it with lookup2(table, rowKey, colKey)",
			)
			continue
		}
		multiErr.Add(
			"expression",
			errortypes.ErrInvalid,
			"Unknown rate table: "+table+
				" — a lookup table is an active rate matrix with a single axis",
		)
	}

	for _, table := range refs.Multi {
		if lookup.Has2(table) {
			continue
		}
		if lookup.Has(table) {
			multiErr.Add(
				"expression",
				errortypes.ErrInvalid,
				"Rate table "+table+
					" has a single axis — address it with lookup(table, key)",
			)
			continue
		}
		multiErr.Add(
			"expression",
			errortypes.ErrInvalid,
			"Unknown rate table: "+table+
				" — lookup2 addresses an active rate matrix with exactly two axes",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) BuildValidationEnvironment(
	schemaID string,
	variables map[string]any,
) (map[string]any, error) {
	env, _, err := s.engine.GetEnvironmentBuilder().
		BuildValidationEnvironment(schemaID, variables)
	return env, err
}

func (s *Service) GetAvailableVariables(schemaID string) []*formulatypes.FieldSource {
	return s.engine.GetEnvironmentBuilder().GetAvailableVariables(schemaID)
}

func (s *Service) GetRequiredPreloads(schemaID string) []string {
	return s.engine.GetEnvironmentBuilder().GetRequiredPreloads(schemaID)
}

func (s *Service) GetEngine() *engine.Engine {
	return s.engine
}

func (s *Service) GetResolver() *resolver.Resolver {
	return s.resolver
}
