package reportingtest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func Tenant(userID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: userID,
	}
}

type Organizations struct {
	tenantports.OrganizationRepository
}

func (Organizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

var ErrNotExecuted = errors.New("the test engine does not execute reports")

var sharedCatalog = sync.OnceValues(reportengine.NewCatalog)

func ReportCatalog(t *testing.T) *analytics.Catalog {
	t.Helper()
	catalog, err := sharedCatalog()
	require.NoError(t, err)
	return catalog
}

type EngineCall struct {
	Kind     string
	Request  analytics.Request
	Cursor   string
	PageSize int
	Row      []ir.DrillValue
	Column   string
	Drill    analytics.DrillOptions
}

type Engine struct {
	catalog *analytics.Catalog
	err     error

	mu    sync.Mutex
	calls []EngineCall
}

func NewEngine(catalog *analytics.Catalog, failure error) *Engine {
	return &Engine{catalog: catalog, err: failure}
}

func (f *Engine) record(call EngineCall) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *Engine) Recorded(kind string) []EngineCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EngineCall, 0, len(f.calls))
	for _, call := range f.calls {
		if call.Kind == kind {
			out = append(out, call)
		}
	}
	return out
}

func (f *Engine) compile(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Compiled, error) {
	if f.err != nil {
		return nil, f.err
	}
	return analytics.Compile(ctx, f.catalog, req.Policy, req.Definition, req.Params,
		analytics.CompileOptions{Location: req.Location, Now: req.Now})
}

func (f *Engine) Catalog() *analytics.Catalog { return f.catalog }

func (f *Engine) Validate(def *ir.Definition) error {
	_, err := ir.Resolve(def, f.catalog, ir.Limits{})
	return err
}

func (f *Engine) Compile(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Compiled, error) {
	f.record(EngineCall{Kind: "compile", Request: *req})
	return f.compile(ctx, req)
}

func (f *Engine) Preview(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Result, error) {
	f.record(EngineCall{Kind: "preview", Request: *req})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, ErrNotExecuted
}

func (f *Engine) Page(
	ctx context.Context,
	req *analytics.Request,
	cursor string,
	size int,
) (*analytics.Result, error) {
	f.record(EngineCall{Kind: "page", Request: *req, Cursor: cursor, PageSize: size})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, ErrNotExecuted
}

func (f *Engine) Drill(
	ctx context.Context,
	req *analytics.Request,
	row []ir.DrillValue,
	column string,
	opts *analytics.DrillOptions,
) (*analytics.Result, error) {
	f.record(EngineCall{Kind: "drill", Request: *req, Row: row, Column: column, Drill: *opts})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, ErrNotExecuted
}

func (f *Engine) Run(ctx context.Context, req *analytics.Request) (*analytics.Stream, error) {
	f.record(EngineCall{Kind: "run", Request: *req})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, ErrNotExecuted
}

type PermissionReader struct {
	permissionports.PermissionEngine

	denied map[string]bool
}

func NewPermissionReader(denied []permission.Resource) *PermissionReader {
	reader := &PermissionReader{denied: make(map[string]bool, len(denied))}
	for _, resource := range denied {
		reader.denied[resource.String()] = true
	}
	return reader
}

func (c *PermissionReader) GetResourcePermissionsBatch(
	_ context.Context,
	_, _ pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	out := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		ops := []permission.Operation{permission.OpRead, permission.OpExport}
		if c.denied[resource] {
			ops = nil
		}
		out[resource] = &permissionports.ResourcePermissionDetail{
			Resource:       resource,
			Operations:     ops,
			DataScope:      permission.DataScopeOrganization,
			MaxSensitivity: permission.SensitivityConfidential,
		}
	}
	return out, nil
}

type FieldGroups struct {
	Groups protectedfield.GroupSet
}

func (g FieldGroups) Visible(
	context.Context,
	*permissionports.RequestActor,
) protectedfield.GroupSet {
	return g.Groups
}
