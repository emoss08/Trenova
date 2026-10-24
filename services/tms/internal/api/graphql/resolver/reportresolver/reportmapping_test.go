package reportresolver

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogEntityReadsInTheRequestersLanguage(t *testing.T) {
	entity := &reportcatalog.Entity{
		Key:         "customer",
		Resource:    permission.ResourceCustomer,
		Label:       "Customer",
		PluralLabel: "Customers",
		Fields: []reportcatalog.Field{{
			Key:   "status",
			Label: "Status",
			EnumValues: []reportcatalog.EnumValue{
				{Value: "Active", Label: "Active"},
			},
		}},
		Edges: []reportcatalog.Edge{{Name: "parent", Label: "Customer"}},
	}
	decided := &reportaccess.Entity{
		Entity:   entity,
		Readable: true,
		Fields:   []reportaccess.Use{{Decision: reportaccess.Available}},
		Edges:    []reportaccess.Use{{Decision: reportaccess.Available}},
	}

	english := catalogEntityToModel(i18n.EN, nil, decided)
	spanish := catalogEntityToModel(i18n.ES, nil, decided)

	assert.Equal(t, "Customer", english.Label)
	require.Len(t, spanish.Fields, 1)
	require.Len(t, spanish.Fields[0].EnumValues, 1)
	require.Len(t, spanish.Edges, 1)

	for source, got := range map[string]string{
		"Customer":  spanish.Label,
		"Customers": spanish.PluralLabel,
		"Status":    spanish.Fields[0].Label,
		"Active":    spanish.Fields[0].EnumValues[0].Label,
	} {
		want := i18n.Translate(i18n.ES, source)
		require.NotEqual(t, source, want, "the Spanish catalog has no entry for %q", source)
		assert.Equal(t, want, got)
	}
	assert.Equal(t, spanish.Label, spanish.Edges[0].Label)
	assert.Equal(t, "customer", spanish.Key)
	assert.Equal(t, "Active", spanish.Fields[0].EnumValues[0].Value)
}

func TestCatalogNamesWhatEachFieldNeeds(t *testing.T) {
	entity := &reportcatalog.Entity{
		Key:         "worker",
		Resource:    permission.ResourceWorker,
		Label:       "Worker",
		PluralLabel: "Workers",
		Fields: []reportcatalog.Field{
			{Key: "firstName", Label: "First Name", Class: reportcatalog.ClassReference},
			{Key: "phoneNumber", Label: "Phone Number"},
			{
				Key:       "licenseNumber",
				Label:     "License Number",
				Protected: permission.FieldGroupPersonal,
			},
		},
		Edges: []reportcatalog.Edge{{Name: "profile", Label: "Profile"}},
	}
	decided := &reportaccess.Entity{
		Entity: entity,
		Fields: []reportaccess.Use{
			{Decision: reportaccess.Available, Reference: true},
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
			{
				Decision: reportaccess.NeedsFieldGroup,
				Group:    permission.FieldGroupPersonal,
			},
		},
		Edges: []reportaccess.Use{
			{Decision: reportaccess.NeedsKindRead, Kind: permission.ResourceWorker},
		},
	}

	model := catalogEntityToModel(i18n.EN, nil, decided)

	assert.False(t, model.Readable)
	require.Len(t, model.Fields, 3)
	assert.Equal(t, gqlmodel.ReportFieldAccessAvailable, model.Fields[0].Access)
	assert.True(t, model.Fields[0].InReferenceView)
	assert.Nil(t, model.Fields[0].Protection)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Fields[1].Access)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsFieldGroup, model.Fields[2].Access)
	require.NotNil(t, model.Fields[2].Protection)
	assert.Equal(t, permission.FieldGroupPersonal, model.Fields[2].Protection.Group)
	assert.False(t, model.Fields[2].Protection.Held)
	require.Len(t, model.Edges, 1)
	assert.Equal(t, gqlmodel.ReportFieldAccessNeedsKindRead, model.Edges[0].Access)
	require.NotNil(t, model.Edges[0].NeedsReadOf)
	assert.Equal(t, "worker", model.Edges[0].NeedsReadOf.Resource)
	assert.Equal(t, "worker", model.Edges[0].NeedsReadOf.Label)

	named := catalogEntityToModel(i18n.EN, reportaccess.New(reportaccess.Params{}), decided)
	assert.Equal(t, "Workers", named.Edges[0].NeedsReadOf.Label)
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
