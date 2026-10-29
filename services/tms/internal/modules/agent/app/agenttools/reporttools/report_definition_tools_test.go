package reporttools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReportWriter struct {
	entries     []*canned.Entry
	definitions []*report.ReportDefinition

	created   *reporting.SaveDefinitionRequest
	updated   *reporting.SaveDefinitionRequest
	forked    *reporting.ForkCannedRequest
	validated *reporting.SaveDefinitionRequest
	invalid   error
}

func (f *fakeReportWriter) ValidateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) error {
	f.validated = req

	return f.invalid
}

func (f *fakeReportWriter) GetCanned(key string) (*canned.Entry, error) {
	for _, entry := range f.entries {
		if entry.Key == key {
			return entry, nil
		}
	}

	return nil, errortypes.NewNotFoundError("no such report")
}

func (f *fakeReportWriter) ForkCanned(
	_ context.Context,
	req *reporting.ForkCannedRequest,
) (*report.ReportDefinition, error) {
	f.forked = req

	return &report.ReportDefinition{ID: pulid.MustNew("rdef_")}, nil
}

func (f *fakeReportWriter) GetDefinition(
	_ context.Context,
	req *reporting.GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	for _, definition := range f.definitions {
		if definition.ID == req.DefinitionID {
			return definition, nil
		}
	}

	return nil, errortypes.NewNotFoundError("ReportDefinition not found")
}

func (f *fakeReportWriter) CreateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	f.created = req

	return &report.ReportDefinition{ID: pulid.MustNew("rdef_")}, nil
}

const testCatalogVersion = "test-catalog"

func (f *fakeReportWriter) NewDefinition(
	req *reporting.SaveDefinitionRequest,
) *report.ReportDefinition {
	return reporting.NewDefinition(req, testCatalogVersion)
}

func (f *fakeReportWriter) ApplyDefinitionSave(
	existing *report.ReportDefinition,
	req *reporting.SaveDefinitionRequest,
) {
	reporting.ApplyDefinitionSave(existing, req, testCatalogVersion)
}

func (f *fakeReportWriter) NewCannedFork(
	entry *canned.Entry,
	req *reporting.ForkCannedRequest,
) *report.ReportDefinition {
	return reporting.NewCannedFork(entry, req, testCatalogVersion)
}

func (f *fakeReportWriter) UpdateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	f.updated = req

	return &report.ReportDefinition{ID: req.DefinitionID}, nil
}

func definitionArgument() map[string]any {
	return map[string]any{
		"entity": "shipment",
		"columns": []any{
			map[string]any{
				"id":        "c1",
				"dimension": map[string]any{"field": "customer.name"},
			},
			map[string]any{
				"id":      "c2",
				"label":   "Revenue",
				"measure": map[string]any{"field": "totalChargeAmount", "agg": "sum"},
			},
		},
		"filters": map[string]any{
			"and": []any{
				map[string]any{"field": "status", "op": "eq", "value": "Completed"},
			},
		},
		"sort": []any{map[string]any{"column": "c2", "desc": true}},
	}
}

func ownedReport(owner pulid.ID) *report.ReportDefinition {
	return &report.ReportDefinition{
		ID:            pulid.MustNew("rdef_"),
		Name:          "Revenue by customer",
		Description:   "Completed revenue per customer.",
		Category:      "Accounting",
		Tags:          []string{"revenue"},
		Kind:          report.DefinitionKindCustom,
		OwnerID:       owner,
		Visibility:    report.VisibilityPrivate,
		Status:        report.DefinitionStatusActive,
		DefaultFormat: report.FormatXLSX,
		Definition: &ir.Definition{
			Version: ir.Version,
			Entity:  "shipment",
			Columns: []ir.Column{{ID: "c1", Dimension: &ir.Dimension{Field: "proNumber"}}},
		},
		Version: 4,
	}
}

/*
The report builder is the product's own way of answering a question nobody
wrote a report for, and a model that can only run what exists sends the person
to go and build one. report_create saves what the model wrote through the same
service the builder uses, so the compiler's field-level authorization governs
the model exactly as it governs the person.
*/
func TestCreateReport_SavesTheDefinitionAsAnActiveCustomReport(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"name":        "Revenue by customer",
		"description": "Completed revenue per customer.",
		"category":    "Accounting",
		"tags":        []any{"revenue", "customers"},
		"definition":  definitionArgument(),
	})

	require.NoError(t, NewCreateReportTool(writer).Execute(t.Context(), params))

	require.NotNil(t, writer.created)
	assert.Equal(t, "Revenue by customer", writer.created.Name)
	assert.Equal(t, "Accounting", writer.created.Category)
	assert.Equal(t, []string{"revenue", "customers"}, writer.created.Tags)
	assert.Equal(t, report.DefinitionStatusActive, writer.created.Status)
	assert.Empty(
		t,
		writer.created.Visibility,
		"the service defaults an unset visibility to private",
	)
	assert.Empty(t, writer.created.DefaultFormat, "the service defaults an unset format")

	definition := writer.created.Definition
	require.NotNil(t, definition)
	assert.Equal(t, ir.Version, definition.Version)
	assert.Equal(t, "shipment", definition.Entity)
	require.Len(t, definition.Columns, 2)
	assert.Equal(t, "customer.name", definition.Columns[0].Dimension.Field)
	assert.Equal(t, ir.AggSum, definition.Columns[1].Measure.Agg)
	require.NotNil(t, definition.Filters)
	require.Len(t, definition.Filters.And, 1)
	assert.Equal(t, "Completed", definition.Filters.And[0].Value)
	require.Len(t, definition.Sort, 1)
}

func TestCreateReport_CarriesTheActorTenantAndPrincipal(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Revenue by customer",
		"definition": definitionArgument(),
	})
	params.Actor.PrincipalType = permissionports.PrincipalTypeUser
	params.Actor.PrincipalID = params.Actor.UserID

	require.NoError(t, NewCreateReportTool(writer).Execute(t.Context(), params))

	assert.Equal(t, params.OrganizationID, writer.created.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, writer.created.TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, writer.created.TenantInfo.UserID)
	assert.Equal(t, params.Actor.UserID, writer.created.Principal.UserID)
	assert.Equal(t, permissionports.PrincipalTypeUser, writer.created.Principal.Type)
}

func TestCreateReport_PassesAnExplicitVisibilityAndFormat(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	require.NoError(
		t,
		NewCreateReportTool(
			writer,
		).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
			"name":          "Revenue by customer",
			"visibility":    "shared",
			"defaultFormat": "xlsx",
			"definition":    definitionArgument(),
		})),
	)

	assert.Equal(t, report.VisibilityShared, writer.created.Visibility)
	assert.Equal(t, report.FormatXLSX, writer.created.DefaultFormat)
}

func TestCreateReport_RequiresADefinitionWithColumns(t *testing.T) {
	t.Parallel()

	for name, definition := range map[string]any{
		"absent":     nil,
		"no columns": map[string]any{"entity": "shipment"},
		"no entity": map[string]any{"columns": []any{
			map[string]any{"id": "c1", "dimension": map[string]any{"field": "proNumber"}},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			writer := &fakeReportWriter{}
			args := map[string]any{"name": "Anything"}
			if definition != nil {
				args["definition"] = definition
			}

			err := NewCreateReportTool(
				writer,
			).Execute(t.Context(), agenttoolstest.ExecuteParams(args))

			require.Error(t, err)
			assert.Nil(t, writer.created, "nothing is saved without a whole definition")
		})
	}
}

func TestCreateReport_RefusesAnUnknownVisibilityOrFormat(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	err := NewCreateReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Revenue by customer",
		"visibility": "public",
		"definition": definitionArgument(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "visibility")

	err = NewCreateReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":          "Revenue by customer",
		"defaultFormat": "docx",
		"definition":    definitionArgument(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "defaultFormat")
	assert.Nil(t, writer.created)
}

func TestCreateReport_BoundsTheTags(t *testing.T) {
	t.Parallel()

	tags := make([]any, 0, maxReportTags+1)
	for i := 0; i <= maxReportTags; i++ {
		tags = append(tags, "tag")
	}

	writer := &fakeReportWriter{}
	err := NewCreateReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Revenue by customer",
		"tags":       tags,
		"definition": definitionArgument(),
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tags")
	assert.Nil(t, writer.created)
}

/*
An adjustment changes what was asked for and nothing else. The tool reads the
report back, lays the given fields over it, and sends the whole thing with the
version it read, so a concurrent edit in the builder is refused by the service
rather than overwritten.
*/
func TestUpdateReport_ChangesOnlyWhatWasSent(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(nil)
	existing := ownedReport(params.Actor.UserID)
	writer := &fakeReportWriter{definitions: []*report.ReportDefinition{existing}}
	params.Params = map[string]any{
		"definitionId": existing.ID.String(),
		"name":         "Revenue by customer, completed",
		"definition":   definitionArgument(),
	}

	require.NoError(t, NewUpdateReportTool(writer).Execute(t.Context(), params))

	require.NotNil(t, writer.updated)
	assert.Equal(t, existing.ID, writer.updated.DefinitionID)
	assert.Equal(t, "Revenue by customer, completed", writer.updated.Name)
	assert.Equal(t, existing.Description, writer.updated.Description)
	assert.Equal(t, existing.Category, writer.updated.Category)
	assert.Equal(t, existing.Tags, writer.updated.Tags)
	assert.Equal(t, existing.Visibility, writer.updated.Visibility)
	assert.Equal(t, existing.Status, writer.updated.Status)
	assert.Equal(t, existing.DefaultFormat, writer.updated.DefaultFormat)
	assert.Equal(t, existing.Version, writer.updated.Version)
	require.Len(
		t, writer.updated.Definition.Columns, 2,
		"the sent definition replaces the old one whole",
	)
}

func TestUpdateReport_KeepsTheDefinitionWhenOnlyMetadataChanges(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(nil)
	existing := ownedReport(params.Actor.UserID)
	writer := &fakeReportWriter{definitions: []*report.ReportDefinition{existing}}
	params.Params = map[string]any{
		"definitionId": existing.ID.String(),
		"visibility":   "shared",
		"status":       "archived",
		"tags":         []any{},
	}

	require.NoError(t, NewUpdateReportTool(writer).Execute(t.Context(), params))

	assert.Same(t, existing.Definition, writer.updated.Definition)
	assert.Equal(t, report.VisibilityShared, writer.updated.Visibility)
	assert.Equal(t, report.DefinitionStatusArchived, writer.updated.Status)
	assert.Empty(t, writer.updated.Tags, "an empty list clears the tags")
	assert.Equal(t, existing.Name, writer.updated.Name)
}

func TestUpdateReport_RefusesACallThatChangesNothing(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(nil)
	existing := ownedReport(params.Actor.UserID)
	writer := &fakeReportWriter{definitions: []*report.ReportDefinition{existing}}
	params.Params = map[string]any{"definitionId": existing.ID.String()}

	err := NewUpdateReportTool(writer).Execute(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to change")
	assert.Nil(t, writer.updated)
}

// The service refuses this too, but in a sentence about ownership; the tool
// says what to do instead, which is what a model can act on.
func TestUpdateReport_RefusesSomeoneElsesReport(t *testing.T) {
	t.Parallel()

	existing := ownedReport(pulid.MustNew("usr_"))
	existing.Visibility = report.VisibilityShared
	writer := &fakeReportWriter{definitions: []*report.ReportDefinition{existing}}

	err := NewUpdateReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"definitionId": existing.ID.String(),
		"name":         "Mine now",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "report_create")
	assert.Nil(t, writer.updated)
}

func TestUpdateReport_RefusesAnEmptyNameOrUnknownStatus(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(nil)
	existing := ownedReport(params.Actor.UserID)
	writer := &fakeReportWriter{definitions: []*report.ReportDefinition{existing}}

	params.Params = map[string]any{"definitionId": existing.ID.String(), "name": "  "}
	err := NewUpdateReportTool(writer).Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")

	params.Params = map[string]any{
		"definitionId": existing.ID.String(),
		"status":       "needs_attention",
	}
	err = NewUpdateReportTool(writer).Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")
	assert.Nil(t, writer.updated)
}

func TestUpdateReport_NamesItsTargetFromTheDefinitionID(t *testing.T) {
	t.Parallel()

	tool, ok := NewUpdateReportTool(&fakeReportWriter{}).(agenttoolports.TargetedTool)
	require.True(t, ok)

	id := pulid.MustNew("rdef_")
	target, found := tool.Target(map[string]any{"definitionId": id.String()})
	require.True(t, found)
	assert.Equal(t, permission.ResourceReport, target.Resource)
	assert.Equal(t, id, target.ID)

	_, found = tool.Target(map[string]any{})
	assert.False(t, found)
}

func TestForkReport_CopiesTheBuiltInReportForTheActor(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{
		entries: []*canned.Entry{{Key: "ar_aging_by_customer", Name: "AR Aging"}},
	}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"reportKey": "ar_aging_by_customer",
		"name":      "AR aging, 60-day buckets",
	})

	require.NoError(t, NewForkReportTool(writer).Execute(t.Context(), params))

	require.NotNil(t, writer.forked)
	assert.Equal(t, "ar_aging_by_customer", writer.forked.CannedKey)
	assert.Equal(t, "AR aging, 60-day buckets", writer.forked.Name)
	assert.Equal(t, params.Actor.UserID, writer.forked.TenantInfo.UserID)
}

func TestForkReport_RefusesAnUnknownKey(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	err := NewForkReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"reportKey": "nope",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "report_list")
	assert.Nil(t, writer.forked)
}

// Saving a report is a write against the report resource, gated the way the
// Reports page gates it. Changing or copying one waits for a person to
// approve; creating one runs on its own only while it stays private, which
// its policy classifies per call.
func TestReportDefinitionTools_DeclareTheirGates(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	tiers := map[string]agent.AutonomyTier{
		"report_create": agent.TierAutoExecute,
		"report_update": agent.TierActWithApproval,
		"report_fork":   agent.TierActWithApproval,
	}
	for _, tool := range []agenttoolports.AgentTool{
		NewCreateReportTool(writer),
		NewUpdateReportTool(writer),
		NewForkReportTool(writer),
	} {
		assert.Equal(t, permission.ResourceReport, tool.Policy().Resource, tool.Name())
		assert.Equal(t, tiers[tool.Name()], tool.Policy().DefaultTier, tool.Name())
		assert.True(t, tool.Policy().Reversible, tool.Name())
	}
	assert.Equal(t, permission.OpCreate, NewCreateReportTool(writer).Policy().Operation)
	assert.Equal(t, permission.OpUpdate, NewUpdateReportTool(writer).Policy().Operation)
	assert.Equal(t, permission.OpCreate, NewForkReportTool(writer).Policy().Operation)
}

func TestReportDefinitionTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Revenue by customer",
		"definition": definitionArgument(),
	})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	err := NewCreateReportTool(writer).Execute(t.Context(), params)

	require.ErrorIs(t, err, toolguard.ErrTenantMismatch)
	assert.Nil(t, writer.created)
}

// The definition is compiled before a proposal exists. A person used to
// approve a card that then failed with "unknown field"; now the model hears
// it first and fixes the call.
func TestCreateAndUpdateReport_ValidateTheDefinitionBeforeProposing(t *testing.T) {
	t.Parallel()

	writer := &fakeReportWriter{invalid: errors.New("unknown field \"foo\" on entity \"shipment\"")}
	create := NewCreateReportTool(writer)
	validator, ok := create.(agenttoolports.ToolValidator)
	require.True(t, ok)

	err := validator.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Lane revenue",
		"definition": definitionArgument(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
	require.NotNil(t, writer.validated)
	assert.Equal(t, "Lane revenue", writer.validated.Name)
	assert.Nil(t, writer.created, "validation saves nothing")

	writer.invalid = nil
	require.NoError(t, validator.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Lane revenue",
		"definition": definitionArgument(),
	})))
}

/*
report_create was sent columns twice, once inside definition and once beside
it. Only definition is read, so the top-level list was dropped without a word
and the report saved with whichever columns definition held. A part of the
definition sent beside it is refused before a proposal exists, naming the part
and where it goes.
*/
func TestCreateAndUpdateReport_RefuseADefinitionPartSentBesideTheDefinition(t *testing.T) {
	t.Parallel()

	topLevel := []any{
		map[string]any{"id": "c9", "dimension": map[string]any{"field": "proNumber"}},
	}

	writer := &fakeReportWriter{}
	create, ok := NewCreateReportTool(writer).(agenttoolports.ToolValidator)
	require.True(t, ok)
	err := create.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Peak Distributing shipments",
		"definition": definitionArgument(),
		"columns":    topLevel,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "\"columns\"")
	assert.Contains(t, err.Error(), "inside definition")
	assert.Nil(t, writer.validated, "nothing is compiled from half a call")

	err = NewCreateReportTool(
		writer,
	).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"name":    "Peak Distributing shipments",
		"entity":  "shipment",
		"columns": topLevel,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "\"entity\", \"columns\"")
	assert.Nil(t, writer.created)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"definitionId": pulid.MustNew("rdef_").String(),
		"filters":      map[string]any{"op": "and"},
	})
	writer.definitions = []*report.ReportDefinition{ownedReport(params.Actor.UserID)}
	params.Params["definitionId"] = writer.definitions[0].ID.String()
	err = NewUpdateReportTool(writer).Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "\"filters\"")
	assert.Nil(t, writer.updated)
}

// The schema says the same thing the refusal does, so a model reading it
// sends the definition once.
func TestCreateReport_DescribesSendingTheDefinitionOnceAndFilteringByID(t *testing.T) {
	t.Parallel()

	tool := NewCreateReportTool(&fakeReportWriter{})

	assert.Contains(t, tool.Description(), "Send the report once")
	assert.Contains(t, tool.Description(), "customer.id")
	assert.NotContains(t, tool.Description(), "\"customer.name\"")
	definition, ok := tool.ParamSchema()["properties"].(map[string]any)["definition"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, definition["description"], "nowhere else")
}
