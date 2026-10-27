package tariff

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/wasmmodule"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"go.uber.org/fx"
)

type moduleCalls interface {
	CheckTariff(ctx context.Context, req *rules.TariffCheckRequest) (*rules.TariffCheck, error)
	TestTariff(ctx context.Context, req *rules.TariffTestRequest) (*rules.TariffTestRun, error)
	ScaffoldTariffTest(
		ctx context.Context,
		req *rules.TariffScaffoldRequest,
	) (*rules.TariffScaffold, error)
	SetTariffName(ctx context.Context, source, name string) (*rules.TariffNamed, error)
	TariffCatalog(ctx context.Context) (rules.RawJSON, error)
	Version(ctx context.Context) (wasmmodule.VersionInfo, error)
}

type defaultModule struct{}

func (defaultModule) CheckTariff(
	ctx context.Context,
	req *rules.TariffCheckRequest,
) (*rules.TariffCheck, error) {
	return rules.CheckTariff(ctx, req)
}

func (defaultModule) TestTariff(
	ctx context.Context,
	req *rules.TariffTestRequest,
) (*rules.TariffTestRun, error) {
	return rules.TestTariff(ctx, req)
}

func (defaultModule) ScaffoldTariffTest(
	ctx context.Context,
	req *rules.TariffScaffoldRequest,
) (*rules.TariffScaffold, error) {
	return rules.ScaffoldTariffTest(ctx, req)
}

func (defaultModule) SetTariffName(
	ctx context.Context,
	source, name string,
) (*rules.TariffNamed, error) {
	return rules.SetTariffName(ctx, source, name)
}

func (defaultModule) TariffCatalog(ctx context.Context) (rules.RawJSON, error) {
	return rules.TariffCatalog(ctx)
}

func (defaultModule) Version(ctx context.Context) (wasmmodule.VersionInfo, error) {
	return rules.Version(ctx)
}

type CheckerParams struct {
	fx.In

	Resolver *TemplateResolver
	Context  tariffports.ContextSource
}

type Checker struct {
	resolver *TemplateResolver
	context  tariffports.ContextSource
	module   moduleCalls
	catalog  *catalogCache
}

var _ tariffports.Checker = (*Checker)(nil)

func NewChecker(p CheckerParams) *Checker {
	module := defaultModule{}
	return &Checker{
		resolver: p.Resolver,
		context:  p.Context,
		module:   module,
		catalog:  newCatalogCache(module),
	}
}

type LibraryMode int

const (
	LibrariesLatest LibraryMode = iota
	LibrariesPinned
)

type SourceCheckRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Kind       rules.TariffKind
	Mode       LibraryMode
	Pins       []tarifftypes.Import
	TablePins  []tarifftypes.TablePin
	Values     *rules.TariffInputValues
}

type SourceCheck struct {
	TenantInfo pagination.TenantInfo
	Check      *rules.TariffCheck
	Context    *rules.TariffContext
	Libraries  LibrarySet
	Imports    []tarifftypes.Import
	Tables     []tarifftypes.TablePin
}

func TablePins(check *rules.TariffCheck, tariffCtx *rules.TariffContext) []tarifftypes.TablePin {
	if check == nil || tariffCtx == nil {
		return []tarifftypes.TablePin{}
	}
	pins := make([]tarifftypes.TablePin, 0, len(check.Tables))
	for _, read := range check.Tables {
		at := tableAt(tariffCtx, read.Code)
		if at < 0 {
			continue
		}
		table := &tariffCtx.Tables[at]
		if table.MatrixID.IsNil() || table.Version <= 0 {
			continue
		}
		pins = append(pins, tarifftypes.TablePin{
			Code:          table.Code,
			RateMatrixID:  table.MatrixID,
			VersionNumber: table.Version,
		})
	}
	slices.SortFunc(pins, func(a, b tarifftypes.TablePin) int {
		return strings.Compare(a.Code, b.Code)
	})
	return pins
}

func (c *SourceCheck) OK() bool {
	return c.Check != nil && c.Check.OK && c.Check.FirstError() == nil
}

func (c *Checker) CheckSource(ctx context.Context, req *SourceCheckRequest) (*SourceCheck, error) {
	libraries, err := c.libraries(ctx, req)
	if err != nil {
		return nil, err
	}

	var tablePins []tarifftypes.TablePin
	if req.Mode == LibrariesPinned {
		tablePins = req.TablePins
	}
	tariffCtx, err := c.contextWith(ctx, &tariffports.ContextRequest{
		TenantInfo: req.TenantInfo,
		Libraries:  libraries.Context(),
		Tables:     tablePins,
	})
	if err != nil {
		return nil, err
	}

	check, err := c.module.CheckTariff(ctx, &rules.TariffCheckRequest{
		Source:  req.Source,
		Kind:    req.Kind,
		Context: tariffCtx,
		Values:  req.Values,
	})
	if err != nil {
		return nil, fmt.Errorf("check tariff source: %w", err)
	}

	return &SourceCheck{
		TenantInfo: req.TenantInfo,
		Check:      check,
		Context:    tariffCtx,
		Libraries:  libraries,
		Imports:    libraries.Pins(check.Imports),
		Tables:     TablePins(check, tariffCtx),
	}, nil
}

func (c *Checker) libraries(ctx context.Context, req *SourceCheckRequest) (LibrarySet, error) {
	if req.Mode == LibrariesPinned {
		return c.resolver.PinnedSet(ctx, req.TenantInfo, req.Pins)
	}
	return c.resolver.LatestLibraries(ctx, req.TenantInfo, req.Source)
}

func (c *Checker) contextFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	libraries []rules.TariffLibrary,
) (*rules.TariffContext, error) {
	return c.contextWith(ctx, &tariffports.ContextRequest{
		TenantInfo: tenantInfo,
		Libraries:  libraries,
	})
}

func (c *Checker) contextWith(
	ctx context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	tariffCtx, err := c.context.Context(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build tariff context: %w", err)
	}
	return tariffCtx, nil
}

func (c *Checker) Check(
	ctx context.Context,
	req *tariffports.CheckRequest,
) (*rules.TariffCheck, error) {
	libraries := req.Libraries
	if libraries == nil {
		latest, err := c.resolver.LatestLibraries(ctx, req.TenantInfo, req.Source)
		if err != nil {
			return nil, err
		}
		libraries = latest.Context()
	}

	tariffCtx, err := c.contextFor(ctx, req.TenantInfo, libraries)
	if err != nil {
		return nil, err
	}

	check, err := c.module.CheckTariff(ctx, &rules.TariffCheckRequest{
		Source:  req.Source,
		Kind:    req.Kind,
		Context: tariffCtx,
		Values:  req.Values,
	})
	if err != nil {
		return nil, fmt.Errorf("check tariff source: %w", err)
	}
	return check, nil
}

func (c *Checker) Context(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	libraries LibrarySet,
) (*rules.TariffContext, error) {
	return c.contextFor(ctx, tenantInfo, libraries.Context())
}

func (c *Checker) TableStamp(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	return c.context.TableStamp(ctx, tenantInfo)
}

type ContextCheckRequest struct {
	Source  string
	Kind    rules.TariffKind
	Context *rules.TariffContext
	Values  *rules.TariffInputValues
}

func (c *Checker) CheckWithContext(
	ctx context.Context,
	req *ContextCheckRequest,
) (*rules.TariffCheck, error) {
	check, err := c.module.CheckTariff(ctx, &rules.TariffCheckRequest{
		Source:  req.Source,
		Kind:    req.Kind,
		Context: req.Context,
		Values:  req.Values,
	})
	if err != nil {
		return nil, fmt.Errorf("check tariff source: %w", err)
	}
	return check, nil
}

func (c *Checker) SetName(ctx context.Context, source, name string) (string, error) {
	named, err := c.module.SetTariffName(ctx, source, name)
	if err != nil {
		return "", fmt.Errorf("set tariff name: %w", err)
	}
	return named.Source, nil
}
