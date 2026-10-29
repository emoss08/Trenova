package reportresolver

import (
	"reflect"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogNamesWhatEachFieldNeeds(t *testing.T) {
	decided := &reportingservice.CatalogEntity{
		View: analytics.EntityView{
			Key:    "worker",
			Label:  "Worker",
			Plural: "Workers",
			Fields: []analytics.FieldView{
				{Key: "firstName", Label: "First Name", Operators: []ir.Op{ir.OpEq}},
				{Key: "phoneNumber", Label: "Phone Number"},
				{Key: "licenseNumber", Label: "License Number"},
			},
			Edges: []analytics.EdgeView{{Name: "profile", Label: "Profile", ToMany: true}},
		},
		Resource: permission.ResourceWorker,
		Fields: []reportaccess.Use{
			{Decision: reportaccess.Available, Reference: true},
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
			{
				Decision: reportaccess.NeedsFieldGroup,
				Group:    protectedfield.Personal,
			},
		},
		Edges: []reportaccess.Use{
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
		},
	}

	model := catalogEntityToModel(t.Context(), nil, decided)

	assert.False(t, model.Readable)
	assert.Equal(t, "worker", model.Resource)
	require.Len(t, model.Fields, 3)
	assert.Equal(t, gqlmodel.ReportFieldAccessAvailable, model.Fields[0].Access)
	assert.Equal(t, []string{"eq"}, model.Fields[0].Operators)
	assert.True(t, model.Fields[0].InReferenceView)
	assert.Nil(t, model.Fields[0].Protection)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Fields[1].Access)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsFieldGroup, model.Fields[2].Access)
	require.NotNil(t, model.Fields[2].Protection)
	assert.Equal(t, protectedfield.Personal, model.Fields[2].Protection.Group)
	assert.False(t, model.Fields[2].Protection.Held)
	require.Len(t, model.Edges, 1)
	assert.True(t, model.Edges[0].ToMany)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Edges[0].Access)
	assert.Nil(t, model.Edges[0].NeedsReadOf, "without a resolver no kind can be named")
}

func TestPreviewCarriesLevelsCursorAndDisplay(t *testing.T) {
	decimals := 2
	display := format.Resolved{
		Style:    "currency",
		Decimals: decimals,
		Labels:   map[string]string{"A": "Active"},
	}
	result := &reportingservice.PreviewResult{
		Columns: []table.Column{
			{ID: "total", Label: "Total", Type: value.Decimal, Display: display},
		},
		Rows:   [][]any{{"1.50"}, {"3.00"}},
		Levels: []table.Level{table.Detail, table.Subtotal(1)},
		Totals: []any{"4.50"},
		Cursor: "next",
	}

	model := reportPreviewToModel(result)

	require.Len(t, model.Columns, 1)
	assert.Equal(t, "currency", model.Columns[0].Display.Style)
	assert.Equal(t, map[string]any{"A": "Active"}, map[string]any(model.Columns[0].Display.Labels))
	assert.Equal(t, []int{0, 1}, model.Levels)
	require.NotNil(t, model.Cursor)
	assert.Equal(t, "next", *model.Cursor)
	assert.Nil(t, model.Rollup)
	assert.Equal(t, []any{"4.50"}, model.Totals)
}

func TestDefinitionInputIsDecodedStrictly(t *testing.T) {
	authCtx := &authctx.AuthContext{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	_, err := saveReportDefinitionRequest(authCtx, &gqlmodel.SaveReportDefinitionInput{
		Name:       "Bad",
		Definition: map[string]any{"entity": "shipment", "columns": []any{}, "bogus": true},
	})
	require.Error(t, err)

	req, err := saveReportDefinitionRequest(authCtx, &gqlmodel.SaveReportDefinitionInput{
		Name: "Good",
		Definition: map[string]any{
			"entity":  "shipment",
			"columns": []any{map[string]any{"id": "n", "measure": map[string]any{"agg": "count"}}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "shipment", req.Definition.Entity)
	assert.Equal(t, ir.Version, req.Definition.Version)
}

func TestUpdateReportScheduleRequestKeepsEveryField(t *testing.T) {
	authCtx := &authctx.AuthContext{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
	columnID := "total"
	timezone := "America/Chicago"
	suppress := true
	yes := true
	threshold := 10.0
	input := &gqlmodel.UpdateReportScheduleInput{
		ID:              pulid.MustNew("rsch_").String(),
		Version:         4,
		DefinitionID:    pulid.MustNew("rdef_").String(),
		CronExpression:  "0 8 * * 1",
		Timezone:        &timezone,
		Formats:         []string{"csv"},
		EmailRecipients: []string{"ops@example.com"},
		EmailAttach:     &yes,
		EmailInline:     &yes,
		NotifyUserIds:   []string{pulid.MustNew("usr_").String()},
		Alert: &gqlmodel.ReportScheduleAlertInput{
			Operator:            "gt",
			Threshold:           3,
			ColumnID:            &columnID,
			Value:               &threshold,
			SuppressWhileFiring: &suppress,
		},
		Enabled: true,
	}

	req, err := updateReportScheduleRequest(authCtx, input)
	require.NoError(t, err)

	assert.Equal(t, input.ID, req.ScheduleID.String())
	assert.Equal(t, int64(4), req.Version)
	assert.Equal(t, input.DefinitionID, req.DefinitionID.String())
	assert.Equal(t, "0 8 * * 1", req.CronExpression)
	assert.Equal(t, timezone, req.Timezone)
	assert.Equal(t, []string{"csv"}, req.Formats)
	assert.Equal(t, []string{"ops@example.com"}, req.EmailRecipients)
	assert.True(t, req.EmailAttach)
	assert.True(t, req.EmailInline)
	require.Len(t, req.NotifyUserIDs, 1)
	assert.Equal(t, input.NotifyUserIds[0], req.NotifyUserIDs[0].String())
	require.NotNil(t, req.Alert)
	assert.Equal(t, report.ScheduleAlert{
		Operator:            dbtype.Operator("gt"),
		Threshold:           3,
		ColumnID:            columnID,
		Value:               &threshold,
		SuppressWhileFiring: true,
	}, *req.Alert)
	assert.True(t, req.Enabled)
}

func TestUpdateReportScheduleInputCarriesEveryCreateField(t *testing.T) {
	var input gqlmodel.UpdateReportScheduleInput
	value := reflect.ValueOf(&input).Elem()
	for i := range value.NumField() {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Pointer:
			field.Set(reflect.New(field.Type().Elem()))
		case reflect.Slice:
			field.Set(reflect.MakeSlice(field.Type(), 1, 1))
		case reflect.String:
			field.SetString("x")
		case reflect.Int:
			field.SetInt(1)
		case reflect.Bool:
			field.SetBool(true)
		default:
			t.Fatalf("set %s (%s) in this test", value.Type().Field(i).Name, field.Kind())
		}
	}

	created := reflect.ValueOf(createReportScheduleInputOf(&input)).Elem()
	for i := range created.NumField() {
		name := created.Type().Field(i).Name
		assert.False(t, created.Field(i).IsZero(),
			"updating a schedule drops %s: copy it in createReportScheduleInputOf", name)
	}
}
