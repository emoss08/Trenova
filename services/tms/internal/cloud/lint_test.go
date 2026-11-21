package cloud_test

import (
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/tenantbindlint"
	"github.com/emoss08/trenova/internal/cloud/api/tenanthandler"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rlslint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var systemScopeAllowed = map[string]string{
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.Lock":                "serialize the applies of one tenant's control plane entry with an advisory lock that belongs to no tenant",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.Get":                 "read one tenant's bookkeeping, a global table of control plane metadata the tenant agent addresses by instance",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.Save":                "record one tenant's bookkeeping, a global table of control plane metadata",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.List":                "page every tenant's bookkeeping for the tenant agent's status report, which covers the whole server",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.ByInstances":         "read the bookkeeping of the tenants a usage answer names, in one statement",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.InstanceOf":          "find the control plane instance of the organization a system email is sent for, to tag it",
	"internal/cloud/adapters/postgres/cloudtenantrepository/repository.go:repository.Record":              "record a tenant's lifecycle in the ledger kept after the tenant is removed; counts and reasons only",
	"internal/cloud/adapters/postgres/tenantpeoplerepository/repository.go:repository.Facts":              "read each listed tenant's first administrator and their invite in one statement for the status report, which spans tenants",
	"internal/cloud/adapters/postgres/tenantpeoplerepository/repository.go:repository.EmailTaken":         "check that an address is free across the server before a tenant is made with it, since addresses are unique across the server; answers only whether it is taken",
	"internal/cloud/adapters/postgres/tenantpeoplerepository/repository.go:repository.Page":               "page a removed or suspended tenant's members and count their memberships elsewhere, which span tenants",
	"internal/cloud/adapters/postgres/tenantpeoplerepository/repository.go:repository.Reassign":           "move a removed tenant's member onto the other organization they belong to before the tenant's rows are deleted",
	"internal/cloud/adapters/postgres/tenantpurgerepository/repository.go:repository.loadPlan":            "read the catalog's tenant tables and foreign keys, which belong to no tenant, to order a removal",
	"internal/cloud/adapters/postgres/tenantpurgerepository/repository.go:repository.exec":                "delete a batch of one removed tenant's rows from a table the catalog named; the statement names the tenant's organization or business unit",
	"internal/cloud/adapters/postgres/tenantpurgerepository/repository.go:repository.DeleteOrganization":  "delete a removed tenant's organization and its managed scope rows, whose cascade removes the append-only audit rows no direct delete may touch",
	"internal/cloud/adapters/postgres/tenantpurgerepository/repository.go:repository.DeleteBusinessUnit":  "delete a removed tenant's business unit once nothing of the tenant references it",
	"internal/cloud/adapters/postgres/tenantpurgerepository/repository.go:repository.Remaining":           "count what is left of one removed tenant in every table the catalog names, to prove its removal complete",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Get":                "read one archive, restore or import command by its id: control plane metadata in a global table",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Create":             "record a command idempotently by its id: control plane metadata in a global table",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Save":               "record a command's state and erase its sealed access when it settles",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Open":               "find a tenant's running commands so two never overlap",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Failures":           "read the failed commands of a page of tenants for the status report, which spans tenants",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Progress":           "record a running command's progress",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.SetGrant":           "keep the server's sealed nightly archive location",
	"internal/cloud/adapters/postgres/cloudcommandrepository/repository.go:repository.Grant":              "read the server's sealed nightly archive location",
	"internal/cloud/adapters/postgres/tenantcopyrepository/repository.go:repository.connect":              "open the archive's own session under the system role: a tenant's archive reads or writes that tenant's rows of every table in one snapshot or one transaction, each statement naming the tenant",
	"internal/cloud/adapters/postgres/tenantcopyrepository/repository.go:repository.PlanFor":              "read the catalog, which belongs to no tenant, to plan an archive",
	"internal/cloud/adapters/postgres/tenantcopyrepository/preflight.go:repository.read":                  "check before an import that its organization, rows, keys, addresses, usernames and sign-in address are not on the server; these are unique across the server and answered only as yes or no",
	"internal/cloud/adapters/postgres/tenantcopyrepository/preflight.go:repository.referenceRows":         "map the reference rows an archive names by natural key (states, jurisdictions) to this server's ids; reference rows belong to no tenant",
	"internal/cloud/adapters/postgres/tenantcopyrepository/preflight.go:repository.EnsureAuditPartitions": "create the empty monthly audit partitions an imported audit trail needs; they belong to no tenant",
	"internal/cloud/adapters/postgres/tenantcopyrepository/trails.go:load.Trails":                         "sign an imported tenant's audit seals and AI audit chain under this server's key inside the archive load, whose own session already runs under the system role, and record where its history came from",
	"internal/cloud/app/tenantlifecycle/commands.go:Service.inTenantTx":                                   "record a tenant's archive, restore or import command in the server's bookkeeping under the tenant's advisory lock",
	"internal/cloud/app/tenantlifecycle/apply.go:Service.Apply":                                           "apply the control plane's entry for one tenant: make its business unit, organization and first administrator before any tenant scope exists, then set its plan",
	"internal/cloud/app/tenantlifecycle/apply.go:Service.refuseApart":                                     "record a refusal of a tenant's entry in the server's bookkeeping after the attempt to make the tenant rolled back",
	"internal/cloud/app/tenantlifecycle/removal.go:Service.Remove":                                        "mark one tenant as being removed in the server's bookkeeping",
	"internal/cloud/app/tenantlifecycle/removal.go:Service.FinishRemoval":                                 "mark one tenant removed in the server's bookkeeping once its rows, documents and keys are gone",
}

var tenantBindAllowed = map[string]string{}

func serviceRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestEveryCloudSystemScopeIsReviewed(t *testing.T) {
	t.Parallel()

	sites, err := rlslint.Scan(serviceRoot(t), filepath.Join("internal", "cloud"))
	require.NoError(t, err)

	seen := make(map[string]struct{}, len(sites))
	unreviewed := make([]string, 0)
	unexplained := make([]string, 0)
	for _, site := range sites {
		seen[site.Key] = struct{}{}
		if _, ok := systemScopeAllowed[site.Key]; !ok {
			unreviewed = append(unreviewed, site.Position+" ("+site.Key+")")
		}
		if site.Reason == "" {
			unexplained = append(unexplained, site.Position)
		}
	}

	stale := make([]string, 0)
	for key, why := range systemScopeAllowed {
		if strings.TrimSpace(why) == "" {
			unexplained = append(unexplained, key+" (allowlist entry)")
		}
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}

	assert.Empty(t, unreviewed,
		"dbscope.WithSystem bypasses tenant isolation; add each new cloud caller to "+
			"systemScopeAllowed with why it must see every tenant:\n%s", strings.Join(unreviewed, "\n"))
	assert.Empty(
		t,
		unexplained,
		"dbscope.WithSystem needs a non-empty string literal or package constant as its reason:\n%s",
		strings.Join(unexplained, "\n"),
	)
	assert.Empty(
		t,
		stale,
		"allowlist entries with no remaining caller:\n%s",
		strings.Join(stale, "\n"),
	)
}

func TestCloudRepositoryMethodsRunInScopedTransactions(t *testing.T) {
	t.Parallel()

	root := serviceRoot(t)
	violations, err := rlslint.ScanRepositories(
		filepath.Join(root, "internal", "cloud", "adapters", "postgres"),
		root,
	)
	require.NoError(t, err)

	found := make([]string, 0, len(violations))
	for _, v := range violations {
		found = append(found, v.Position+" "+v.Key+": "+v.Problem)
	}
	assert.Empty(t, found,
		"cloud repository methods must run their statements in a scoped transaction "+
			"(see docs/engineering/row-level-security.md):\n%s", strings.Join(found, "\n"))
}

var tenantFieldNames = []string{
	"OrganizationID",
	"OrgID",
	"BusinessUnitID",
	"BuID",
	"CurrentOrganizationID",
}

func TestTenantRouteBodiesCarryNoTenant(t *testing.T) {
	t.Parallel()

	for _, body := range []any{
		cloudtenant.Desired{},
		tenantlifecycle.TenantUsage{},
		tenanthandler.EntryBody{},
		tenanthandler.ArchiveBody{},
		tenanthandler.RestoreBody{},
		tenanthandler.StorageBody{},
	} {
		for _, path := range tenantFields(reflect.TypeOf(body), "", 0) {
			t.Errorf("%T carries %s: a managed listener route names a tenant by its control plane "+
				"instance, never by an organization or business unit from the request", body, path)
		}
	}
}

func tenantFields(typ reflect.Type, prefix string, depth int) []string {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || depth > 4 {
		return nil
	}
	out := make([]string, 0)
	for i := range typ.NumField() {
		field := typ.Field(i)
		if slices.Contains(tenantFieldNames, field.Name) {
			out = append(out, prefix+field.Name)
		}
		out = append(out, tenantFields(field.Type, prefix+field.Name+".", depth+1)...)
	}
	return out
}

func TestCloudHandlersNeverBindATenantFieldFromTheRequest(t *testing.T) {
	t.Parallel()

	violations, err := tenantbindlint.Check(serviceRoot(t), "./internal/cloud/...")
	require.NoError(t, err)

	unexpected := make([]string, 0)
	seen := make(map[string]struct{}, len(violations))
	for _, v := range violations {
		seen[v.Key()] = struct{}{}
		if _, ok := tenantBindAllowed[v.Key()]; ok {
			continue
		}
		unexpected = append(unexpected, v.String())
	}
	require.Empty(t, unexpected, "\n"+strings.Join(unexpected, "\n"))

	stale := make([]string, 0)
	for key := range tenantBindAllowed {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}
	require.Empty(t, stale, "remove allowlist entries that no longer match a bind site")
}
