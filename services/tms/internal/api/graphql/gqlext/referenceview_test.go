package gqlext

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/api/graphql/selfscoped"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

const repoRootFromHere = "../../../../../.."

// notAKind lists the fields named for a kind whose type is not that kind's
// record, by the reason each is not.
var notAKind = map[string]string{
	"AIProvider.apiKey": "the provider's own key status, not an API key record",
	"Customer.emailProfile": "the customer's own delivery settings, part of the customer, " +
		"not an email profile record",
	"Shipment.formulaTemplate": "its resolver shows the template only with formula_template " +
		"read (formulavisibility), so the screen never sees it otherwise",
	"AgentAccessPreview.roles": "a row about the agent's audience; its role field is a Role, " +
		"gated as one",
	"AgentAudienceSuggestion.roles": "a row about the agent's audience; its role field is a Role, " +
		"gated as one",
	"DeskMemorySettings.roles": "the memory settings' own list of which roles may write, by " +
		"name",
	"EdiPartnerCard.carriers": "the partner's carrier mapping rows: each carrier's name, code " +
		"and SCAC, which the carrier reference view shows",
	"FuelPurchaseImportBatch.document": "the import's own source file metadata",
	"ReportRunDeskCard.report": "the run's own report: its name, description and category, " +
		"read with the run under report read",
	"LateChargeAssessmentResult.customers": "the result of the caller's own late-charge run, " +
		"one row per customer it assessed",
	"ShipmentBillingWatch.customers": "an aggregate of shipments by customer on the shipment " +
		"analytics page, which applies reference views itself",
	"ShipmentCapacity.carriers": "counts of postings and tenders, with no carrier record",
	"ShipmentCoverageSuggestions.carriers": "a coverage suggestion for the move: the carrier's " +
		"name and MC number, which the carrier reference view shows, and the move's own quote",
	"ShipmentWatchlist.detention": "detention totals on the shipment analytics page, not a " +
		"detention record",
	"TriageCustomerPeek.shipments": "its resolver includes shipments only for a caller who can " +
		"read shipments",
	"WorkerCredential.document":             "the credential's own attachment metadata",
	"WorkerEmploymentVerification.document": "the verification's own attachment metadata",
	"WorkerLeaveCase.document":              "the case's own attachment metadata",
	"WorkerTrainingRecord.document":         "the record's own attachment metadata",
	"WorkerDOTTest.document":                "the test's own attachment metadata",
	"WorkerEmploymentEvent.document":        "the event's own attachment metadata",
	"WorkerInjury.document":                 "the case's own attachment metadata",
	"WorkerSafetyEvent.document":            "the event's own attachment metadata",
}

func loadSchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	return schema
}

// Every field a kind's reference view names exists on the kind's GraphQL
// type or on a projection of it, so a typo cannot silently show nothing. A
// kind no GraphQL type shows, such as a document, is checked against the REST
// types that carry it (responseguard).
func TestEveryReferenceFieldExistsOnItsKind(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)

	types := make(map[permission.Resource][]*ast.Definition)
	for name, kind := range kinds {
		types[kind] = append(types[kind], schema.Types[name])
	}
	var missing []string
	for _, def := range registry.All() {
		if len(types[permission.Resource(def.Resource)]) == 0 {
			continue
		}
		for _, field := range def.ReferenceView {
			found := false
			for _, typ := range types[permission.Resource(def.Resource)] {
				if typ.Fields.ForName(field) != nil {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, def.Resource+"."+field)
			}
		}
	}
	slices.Sort(missing)
	assert.Empty(t, missing,
		"reference view fields that no GraphQL type of their kind has. Fix the field name in the "+
			"kind's ReferenceView (resources_*.go), or add @kind(resource:) to the projection that "+
			"carries it:\n  %s", strings.Join(missing, "\n  "))
}

// A field named for a kind that returns some other type is a projection of
// that kind, and must say so with @kind, or the guard cannot gate it.
func TestEveryKindToKindFieldIsCovered(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)

	var uncovered []string
	for name, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn || isRoot(name) {
			continue
		}
		for _, field := range def.Fields {
			child := schema.Types[field.Type.Name()]
			if child == nil || child.Kind != ast.Object {
				continue
			}
			if _, kind := kinds[child.Name]; kind {
				continue
			}
			named := stringutils.ConvertCamelToSnake(field.Name)
			if !registry.HasResource(named) &&
				!registry.HasResource(strings.TrimSuffix(named, "s")) {
				continue
			}
			key := name + "." + field.Name
			if _, listed := notAKind[key]; !listed {
				uncovered = append(uncovered, fmt.Sprintf("%s returns %s", key, child.Name))
			}
		}
	}
	slices.Sort(uncovered)
	assert.Empty(t, uncovered,
		"fields named for a kind that return a type with no kind. Add @kind(resource: \"<kind>\") "+
			"to the returned type so the field guard gates it as that kind, or list the field in "+
			"notAKind with the reason it is not that kind's record:\n  %s",
		strings.Join(uncovered, "\n  "))
}

// Every client operation that selects a field outside the reference view of
// a linked record declares that it needs the kind's read, so the screen knows
// to draw the field's withheld state.
func TestEveryLinkedFieldIsDeclared(t *testing.T) {
	t.Parallel()

	var undeclared []string
	for _, field := range linkedFields(t) {
		if field.Declared {
			continue
		}
		undeclared = append(undeclared, fmt.Sprintf(
			"%s: it is not in the %s reference view. Select ...%s instead, or add "+
				"@requiresRead(kind: %q) to the operation and draw the field's withheld state",
			field, field.Kind, ReferenceFragmentName(field.Type), field.Kind,
		))
	}
	assert.Empty(t, undeclared, "%s", strings.Join(slices.Compact(undeclared), "\n"))
}

// A field a declared operation selects is answered null without the kind's
// read, so it must be nullable; a non-null one would null its parent.
func TestEveryDeclaredLinkedFieldIsNullable(t *testing.T) {
	t.Parallel()

	var nonNull []string
	for _, field := range linkedFields(t) {
		if field.Declared && field.NonNull {
			nonNull = append(nonNull, fmt.Sprintf(
				"%s.%s is selected under @requiresRead(kind: %q) by %s: make it nullable",
				field.Type, field.Field, field.Kind, field.Operation))
		}
	}
	slices.Sort(nonNull)
	assert.Empty(t, slices.Compact(nonNull), "%s", strings.Join(slices.Compact(nonNull), "\n"))
}

func TestReferenceFragmentsAreGenerated(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	fragments, err := RenderReferenceFragments(schema, registry)
	require.NoError(t, err)
	views, err := RenderReferenceViews(schema, registry)
	require.NoError(t, err)

	for path, want := range map[string]string{
		ReferenceFragmentsPath: fragments,
		ReferenceViewsPath:     views,
	} {
		got, err := os.ReadFile(filepath.Join(repoRootFromHere, path))
		require.NoError(t, err)
		assert.Equal(
			t,
			want,
			string(got),
			"%s is out of date: run `go run ./internal/api/graphql/gqlext/refgen` from services/tms, "+
				"then `pnpm --filter @trenova/graphql codegen`",
			path,
		)
	}
	assert.NotContains(t, fragments, "{\n}",
		"a kind type with none of its view's fields, such as a summary, has no fragment")
}

// A summary that carries records of its kind but holds neither an id nor a
// field of the kind's view is never linked by itself, so it has no fragment;
// an empty one is not GraphQL.
func TestReferenceFragmentsLeaveOutASummaryWithNothingToShow(t *testing.T) {
	t.Parallel()

	fragments, err := RenderReferenceFragments(loadSchema(t), fieldsensitivity.DefaultRegistry())
	require.NoError(t, err)

	assert.NotContains(t, fragments, "on FuelIndexLatestPrice")
	assert.Contains(t, fragments, "on FuelSurchargeProgram")
	assert.NotContains(t, fragments, "{\n}")
}

var (
	resolverFunc   = regexp.MustCompile(`func \(r \*(Query|Mutation)Resolver\) (\w+)\(`)
	resolverCheck  = regexp.MustCompile(`permission\.(Resource\w+),\s*permission\.(Op\w+)`)
	resourceConst  = regexp.MustCompile(`(Resource\w+)\s+Resource\s*=\s*"([a-z_0-9]+)"`)
	operationConst = regexp.MustCompile(`(Op\w+)\s+Operation\s*=\s*"([a-z_0-9]+)"`)
)

// A root field's @authorizes declaration is held to its resolver: the
// resolver checks the kind with read or an operation that implies it.
func TestEveryAuthorizesDeclarationIsCheckedByItsResolver(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	checks := resolverReadChecks(t)

	var unchecked []string
	for _, root := range []*ast.Definition{schema.Query, schema.Mutation} {
		for _, field := range root.Fields {
			for _, directive := range field.Directives.ForNames(authorizesDirective) {
				resource := directive.Arguments.ForName(kindResourceArgument).Value.Raw
				if !slices.Contains(checks[strings.ToLower(root.Name+"."+field.Name)], resource) {
					unchecked = append(unchecked, fmt.Sprintf("%s.%s @authorizes(resource: %q)",
						root.Name, field.Name, resource))
				}
			}
		}
	}
	slices.Sort(unchecked)
	assert.Empty(t, unchecked,
		"root fields that declare @authorizes for a kind their resolver does not check with read "+
			"(or an operation implying it). Check it in the resolver, or remove the declaration so "+
			"the records are gated as linked ones:\n  %s", strings.Join(unchecked, "\n  "))
}

// resolverReadChecks is, for each root resolver ("query.shipments"), the
// kinds its body checks with read or an operation implying it.
func resolverReadChecks(t *testing.T) map[string][]string {
	t.Helper()

	constants := make(map[string]string)
	raw, err := os.ReadFile("../../../modules/identity/domain/permission/resource_gen.go")
	require.NoError(t, err)
	for _, m := range resourceConst.FindAllStringSubmatch(string(raw), -1) {
		constants[m[1]] = m[2]
	}
	operations := make(map[string]permission.Operation)
	raw, err = os.ReadFile("../../../modules/identity/domain/permission/operations.go")
	require.NoError(t, err)
	for _, m := range operationConst.FindAllStringSubmatch(string(raw), -1) {
		operations[m[1]] = permission.Operation(m[2])
	}
	checks := make(map[string][]string)
	files, err := filepath.Glob("../resolver/*/*.resolvers.go")
	require.NoError(t, err)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		src := string(raw)
		for _, loc := range resolverFunc.FindAllStringSubmatchIndex(src, -1) {
			end := strings.Index(src[loc[1]:], "\nfunc ")
			body := src[loc[1]:]
			if end >= 0 {
				body = body[:end]
			}
			key := strings.ToLower(src[loc[2]:loc[3]] + "." + src[loc[4]:loc[5]])
			for _, m := range resolverCheck.FindAllStringSubmatch(body, -1) {
				op := operations[m[2]]
				if slices.Contains(
					permission.Implied([]permission.Operation{op}),
					permission.OpRead,
				) {
					checks[key] = append(checks[key], constants[m[1]])
				}
			}
		}
	}

	return checks
}

// rootKinds is every kind a root field returns as its own record: the type
// it returns, or the node of the connection it returns.
func rootKinds(
	schema *ast.Schema,
	kinds map[string]permission.Resource,
	def *ast.Definition,
	seen map[string]bool,
) []permission.Resource {
	if def == nil || seen[def.Name] {
		return nil
	}
	seen[def.Name] = true
	if kind, ok := kinds[def.Name]; ok {
		return []permission.Resource{kind}
	}
	if !strings.HasSuffix(def.Name, "Connection") && !strings.HasSuffix(def.Name, "Edge") {
		return nil
	}
	var out []permission.Resource
	for _, field := range def.Fields {
		out = append(out, rootKinds(schema, kinds, schema.Types[field.Type.Name()], seen)...)
	}
	return out
}

// Row-level security does not hide a shipment, order, invoice or worker the
// person cannot read: a record that links to one shows its reference view,
// so the database must return it. A root field returns its records whole, so
// its resolver must check read on the kind (or an operation implying it)
// before it reads; nothing else would stop it.
func TestEveryRootOfADatabaseScopedKindChecksItsRead(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := permissioncatalog.NewRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)
	scoped := registry.DatabaseScopedResources()
	checks := resolverReadChecks(t)

	var unchecked []string
	reached := 0
	for _, root := range []*ast.Definition{schema.Query, schema.Mutation, schema.Subscription} {
		if root == nil {
			continue
		}
		for _, field := range root.Fields {
			if selfscoped.Root(root.Name, field.Name) {
				continue
			}
			for _, kind := range rootKinds(schema, kinds, schema.Types[field.Type.Name()], map[string]bool{}) {
				if !slices.Contains(scoped, kind.String()) {
					continue
				}
				reached++
				if !slices.Contains(
					checks[strings.ToLower(root.Name+"."+field.Name)],
					kind.String(),
				) {
					unchecked = append(
						unchecked,
						fmt.Sprintf("%s.%s returns %s", root.Name, field.Name, kind),
					)
				}
			}
		}
	}
	require.Positive(t, reached)
	slices.Sort(unchecked)
	assert.Empty(t, unchecked,
		"root fields that return records of a database-scoped kind without checking read on it "+
			"(or an operation implying it) in the resolver:\n  %s", strings.Join(unchecked, "\n  "))
}
