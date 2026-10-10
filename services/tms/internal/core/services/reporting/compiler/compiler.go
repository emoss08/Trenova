package compiler

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ reportports.ReportCompiler = (*Compiler)(nil)

type Params struct {
	fx.In

	Config           *config.Config
	Logger           *zap.Logger
	PermissionEngine permissionports.PermissionEngine
}

type Compiler struct {
	catalog            *reportcatalog.Catalog
	permissionEngine   permissionports.PermissionEngine
	permissionRegistry *permission.Registry
	limits             limits
	l                  *zap.Logger
}

func New(p Params) *Compiler {
	reporting := p.Config.GetReportingConfig()

	return &Compiler{
		catalog:            &reportcatalog.Default,
		permissionEngine:   p.PermissionEngine,
		permissionRegistry: permission.NewRegistry(),
		limits: limits{
			maxToOneJoins:       reporting.GetMaxToOneJoins(),
			maxToManySubqueries: reporting.GetMaxToManySubqueries(),
			maxDimensions:       reporting.GetMaxDimensions(),
			maxPivotColumns:     reporting.GetMaxPivotColumns(),
			maxPathDepth:        reporting.GetMaxPathDepth(),
			maxLimit:            reporting.GetMaxDefinitionLimit(),
		},
		l: p.Logger.Named("service.reporting-compiler"),
	}
}

func NewWithCatalog(
	catalog *reportcatalog.Catalog,
	engine permissionports.PermissionEngine,
	registry *permission.Registry,
	reporting *config.ReportingConfig,
	logger *zap.Logger,
) *Compiler {
	return &Compiler{
		catalog:            catalog,
		permissionEngine:   engine,
		permissionRegistry: registry,
		limits: limits{
			maxToOneJoins:       reporting.GetMaxToOneJoins(),
			maxToManySubqueries: reporting.GetMaxToManySubqueries(),
			maxDimensions:       reporting.GetMaxDimensions(),
			maxPivotColumns:     reporting.GetMaxPivotColumns(),
			maxPathDepth:        reporting.GetMaxPathDepth(),
			maxLimit:            reporting.GetMaxDefinitionLimit(),
		},
		l: logger.Named("service.reporting-compiler"),
	}
}

func (c *Compiler) ValidateAndAuthorize(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
) (*reportports.ReportValidationResult, error) {
	compiled, err := c.compile(ctx, req, false)
	if err != nil {
		return nil, err
	}

	return &reportports.ReportValidationResult{
		ReferencedEntities: compiled.ReferencedEntities,
		Columns:            compiled.Columns,
		Complexity:         compiled.Complexity,
	}, nil
}

func (c *Compiler) Compile(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
) (*reportports.CompiledReportQuery, error) {
	return c.compile(ctx, req, true)
}

// CompileForPreview compiles with read-level authorization only: previews
// render inside the builder and never produce an export artifact, so OpExport
// is not required.
func (c *Compiler) CompileForPreview(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
) (*reportports.CompiledReportQuery, error) {
	return c.compile(ctx, req, false)
}

func (c *Compiler) compile(
	ctx context.Context,
	req *reportports.ReportCompileRequest,
	requireExport bool,
) (*reportports.CompiledReportQuery, error) {
	v, err := c.validate(req)
	if err != nil {
		return nil, err
	}

	az, err := c.authorize(ctx, req, v, requireExport)
	if err != nil {
		return nil, err
	}

	plan, err := c.planJoins(v)
	if err != nil {
		return nil, err
	}

	return c.emit(i18n.FromContext(ctx), req, v, plan, az)
}

func (c *Compiler) complexity(v *validatedDef, plan *joinPlan) reportports.ReportComplexity {
	dims, measures := 0, 0
	for i := range v.columns {
		if v.columns[i].spec.Kind == report.ColumnKindDimension {
			dims++
		} else {
			measures++
		}
	}

	pivotColumns := 0
	if v.def.Pivot != nil {
		pivotColumns = len(v.def.Pivot.Values) * len(v.def.Pivot.MeasureIDs)
		if v.def.Pivot.IncludeOther {
			pivotColumns += len(v.def.Pivot.MeasureIDs)
		}
	}

	score := len(plan.joins) + 2*len(plan.laterals) + dims + pivotColumns/10

	return reportports.ReportComplexity{
		ToOneJoins:       len(plan.joins),
		ToManySubqueries: len(plan.laterals),
		Dimensions:       dims,
		Measures:         measures,
		PivotColumns:     pivotColumns,
		Score:            score,
	}
}
