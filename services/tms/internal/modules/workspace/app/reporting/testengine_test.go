package reporting

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotExecuted = errors.New("the test engine does not execute reports")

var sharedCatalog = sync.OnceValues(reportengine.NewCatalog)

func testCatalog(t *testing.T) *analytics.Catalog {
	t.Helper()
	catalog, err := sharedCatalog()
	require.NoError(t, err)
	return catalog
}

type engineCall struct {
	Kind     string
	Request  analytics.Request
	Cursor   string
	PageSize int
	Row      []ir.DrillValue
	Column   string
	Drill    analytics.DrillOptions
}

type fakeEngine struct {
	catalog *analytics.Catalog
	err     error

	mu    sync.Mutex
	calls []engineCall
}

func (f *fakeEngine) record(call engineCall) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakeEngine) recorded(kind string) []engineCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]engineCall, 0, len(f.calls))
	for _, call := range f.calls {
		if call.Kind == kind {
			out = append(out, call)
		}
	}
	return out
}

func (f *fakeEngine) compile(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Compiled, error) {
	if f.err != nil {
		return nil, f.err
	}
	return analytics.Compile(ctx, f.catalog, req.Policy, req.Definition, req.Params,
		analytics.CompileOptions{Location: req.Location, Now: req.Now})
}

func (f *fakeEngine) Catalog() *analytics.Catalog { return f.catalog }

func (f *fakeEngine) Compile(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Compiled, error) {
	f.record(engineCall{Kind: "compile", Request: *req})
	return f.compile(ctx, req)
}

func (f *fakeEngine) Preview(
	ctx context.Context,
	req *analytics.Request,
) (*analytics.Result, error) {
	f.record(engineCall{Kind: "preview", Request: *req})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, errNotExecuted
}

func (f *fakeEngine) Page(
	ctx context.Context,
	req *analytics.Request,
	cursor string,
	size int,
) (*analytics.Result, error) {
	f.record(engineCall{Kind: "page", Request: *req, Cursor: cursor, PageSize: size})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, errNotExecuted
}

func (f *fakeEngine) Count(ctx context.Context, req *analytics.Request) (int64, error) {
	f.record(engineCall{Kind: "count", Request: *req})
	if _, err := f.compile(ctx, req); err != nil {
		return 0, err
	}
	return 0, errNotExecuted
}

func (f *fakeEngine) Drill(
	ctx context.Context,
	req *analytics.Request,
	row []ir.DrillValue,
	column string,
	opts *analytics.DrillOptions,
) (*analytics.Result, error) {
	f.record(engineCall{Kind: "drill", Request: *req, Row: row, Column: column, Drill: *opts})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, errNotExecuted
}

func (f *fakeEngine) Run(ctx context.Context, req *analytics.Request) (*analytics.Stream, error) {
	f.record(engineCall{Kind: "run", Request: *req})
	if _, err := f.compile(ctx, req); err != nil {
		return nil, err
	}
	return nil, errNotExecuted
}

type confidentialReader struct {
	permissionports.PermissionEngine

	denied map[string]bool
}

func (c *confidentialReader) GetResourcePermissionsBatch(
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

type groupAccess struct {
	visible permission.FieldGroupSet
}

func (g groupAccess) Visible(
	context.Context,
	*permissionports.RequestActor,
) permission.FieldGroupSet {
	return g.visible
}

type engineOptions struct {
	denied  []permission.Resource
	groups  permission.FieldGroupSet
	failure error
}

func engineService(t *testing.T, opts engineOptions) (*Service, *fakeEngine) {
	t.Helper()
	catalog := testCatalog(t)
	reader := &confidentialReader{denied: make(map[string]bool, len(opts.denied))}
	for _, resource := range opts.denied {
		reader.denied[resource.String()] = true
	}
	registry := permission.NewRegistry()
	access, err := reportaccess.New(reportaccess.Params{
		Catalog:  catalog,
		Engine:   reader,
		Registry: registry,
		Views:    refview.NewPort(refview.New(registry), reader),
		Fields:   groupAccess{visible: opts.groups},
	})
	require.NoError(t, err)
	engine := &fakeEngine{catalog: catalog, err: opts.failure}
	return &Service{
		l:       zap.NewNop(),
		engine:  engine,
		perms:   reader,
		access:  access,
		orgRepo: previewOrganizations{},
		cfg:     &config.ReportingConfig{PreviewRowLimit: 40},
	}, engine
}

func storedRunDefinition() *ir.Definition {
	return &ir.Definition{
		Version: ir.Version,
		Entity:  "shipment",
		Columns: []ir.Column{{
			ID:        "c_customer",
			Dimension: &ir.Dimension{Field: "customer.name"},
		}},
	}
}

func testUserID() pulid.ID {
	return pulid.MustNew("usr_")
}
