package toolpreview

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResource = permission.Resource("preview_test_record")

type testCustomer struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type testRecord struct {
	ID             pulid.ID        `json:"id"`
	OrganizationID pulid.ID        `json:"organizationId"`
	Version        int64           `json:"version"`
	ProNumber      string          `json:"proNumber"`
	Status         string          `json:"status"`
	RatingAmount   decimal.Decimal `json:"ratingAmount"`
	CustomerID     pulid.ID        `json:"customerId"`
	BankAccount    string          `json:"bankAccount"`
	Notes          string          `json:"notes"`
	PickupAt       int64           `json:"pickupAt"`
	Customer       *testCustomer   `json:"customer,omitempty"`
	UpdatedAt      int64           `json:"updatedAt"`
}

func testRegistry(t *testing.T) *permission.Registry {
	t.Helper()

	registry := permission.NewEmptyRegistry()
	require.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:           testResource.String(),
		DefaultSensitivity: permission.SensitivityInternal,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"bankAccount":  permission.SensitivityConfidential,
			"ratingAmount": permission.SensitivityRestricted,
		},
	}))

	return registry
}

func sampleRecord() *testRecord {
	return &testRecord{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        4,
		ProNumber:      "PRO-100",
		Status:         "New",
		RatingAmount:   decimal.RequireFromString("40000.00"),
		CustomerID:     pulid.MustNew("cus_"),
		BankAccount:    "000123",
		Notes:          "",
		PickupAt:       1767225600,
		Customer:       &testCustomer{ID: pulid.MustNew("cus_"), Name: "Acme"},
		UpdatedAt:      1767225600,
	}
}

func fieldByPath(t *testing.T, change *agent.RecordChange, path string) agent.PreviewFieldChange {
	t.Helper()

	for _, field := range change.Fields {
		if field.Path == path {
			return field
		}
	}
	require.Failf(t, "field not found", "no field %q in %+v", path, change.Fields)

	return agent.PreviewFieldChange{}
}

func paths(change *agent.RecordChange) []string {
	out := make([]string, 0, len(change.Fields))
	for _, field := range change.Fields {
		out = append(out, field.Path)
	}

	return out
}

func TestUpdate_DiffsWhatThePlanFunctionChanges(t *testing.T) {
	t.Parallel()

	before := sampleRecord()
	newCustomer := pulid.MustNew("cus_")
	version := before.Version

	change, err := Update(
		Record{Resource: testResource, ID: before.ID, Version: &version},
		before,
		func(r *testRecord) error {
			r.Status = "Cancelled"
			r.RatingAmount = decimal.RequireFromString("42000.50")
			r.CustomerID = newCustomer
			r.BankAccount = "999"
			r.Notes = "Customer pulled the load"
			r.Version++
			r.UpdatedAt = 1767229200
			r.Customer = &testCustomer{ID: newCustomer, Name: "Other"}
			return nil
		},
		WithRefs(map[string]permission.Resource{"customerId": permission.ResourceCustomer}),
		WithRegistry(testRegistry(t)),
	)
	require.NoError(t, err)

	assert.Equal(t, agent.PreviewOperationUpdate, change.Operation)
	assert.Equal(t, "PRO-100", change.Label, "the record is named by its PRO")
	assert.Equal(t, before.ID, change.EntityID)
	assert.Equal(t, []string{"status", "ratingAmount", "customerId", "notes"}, paths(change),
		"declared order; version, updatedAt, the relation and the confidential field are dropped")

	status := fieldByPath(t, change, "status")
	assert.Equal(t, assistantartifact.DisplayStatus, status.Type)
	assert.Equal(t, "New", status.Before)
	assert.Equal(t, "Cancelled", status.After)
	assert.Equal(t, "Status", status.Label)

	amount := fieldByPath(t, change, "ratingAmount")
	assert.Equal(t, assistantartifact.DisplayMoney, amount.Type)
	assert.Equal(t, "40000", amount.Before)
	assert.Equal(t, "42000.5", amount.After)
	assert.Equal(t, permission.SensitivityRestricted, amount.Sensitivity)

	customer := fieldByPath(t, change, "customerId")
	require.NotNil(t, customer.BeforeRef)
	require.NotNil(t, customer.AfterRef)
	assert.Equal(t, permission.ResourceCustomer, customer.AfterRef.Resource)
	assert.Equal(t, newCustomer, customer.AfterRef.ID)
	assert.Equal(t, before.CustomerID, customer.BeforeRef.ID)

	notes := fieldByPath(t, change, "notes")
	assert.Nil(t, notes.Before, "an empty value is nothing, not an empty string")
	assert.Equal(t, "Customer pulled the load", notes.After)

	assert.Equal(t, "New", before.Status, "the record handed in is never changed")
}

func TestUpdate_ReturnsThePlanFunctionsRefusal(t *testing.T) {
	t.Parallel()

	refusal := errors.New("a delivered shipment cannot be cancelled")
	_, err := Update(Record{Resource: testResource}, sampleRecord(), func(*testRecord) error {
		return refusal
	})

	require.ErrorIs(t, err, refusal)

	_, err = Update[testRecord](Record{Resource: testResource}, nil, nil)
	require.ErrorIs(t, err, ErrNoRecord)
}

func TestUpdate_VolatileAndIgnoredPaths(t *testing.T) {
	t.Parallel()

	change, err := Update(
		Record{Resource: testResource},
		sampleRecord(),
		func(r *testRecord) error {
			r.Status = "Assigned"
			r.PickupAt = 1767312000
			r.Notes = "call ahead"
			return nil
		},
		Volatile("pickupAt"),
		Ignore("notes"),
		WithRegistry(testRegistry(t)),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"status", "pickupAt"}, paths(change))
	assert.True(t, fieldByPath(t, change, "pickupAt").Volatile)
	assert.Equal(t, assistantartifact.DisplayDateTime, fieldByPath(t, change, "pickupAt").Type)
}

func TestArchive_IsAnUpdateThatRetires(t *testing.T) {
	t.Parallel()

	change, err := Archive(
		Record{Resource: testResource},
		sampleRecord(),
		func(r *testRecord) error {
			r.Status = "Inactive"
			return nil
		},
	)
	require.NoError(t, err)

	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, []string{"status"}, paths(change))
}

func TestCreate_ShowsTheValuesTheRecordWouldHold(t *testing.T) {
	t.Parallel()

	record := sampleRecord()
	change, err := Create(Record{Resource: testResource, Label: "New shipment"}, record,
		WithRegistry(testRegistry(t)),
		WithRefs(map[string]permission.Resource{"customerId": permission.ResourceCustomer}),
	)
	require.NoError(t, err)

	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "New shipment", change.Label)
	assert.ElementsMatch(t,
		[]string{"proNumber", "status", "ratingAmount", "customerId", "pickupAt"},
		paths(change),
	)
	for _, field := range change.Fields {
		assert.Nil(t, field.Before, field.Path)
	}

	only, err := Create(Record{Resource: testResource}, record, Only("proNumber", "status"))
	require.NoError(t, err)
	assert.Equal(t, []string{"proNumber", "status"}, paths(only))
}

func TestDelete_ShowsWhatWouldGo(t *testing.T) {
	t.Parallel()

	change, err := Delete(Record{Resource: testResource}, sampleRecord(), Only("proNumber"))
	require.NoError(t, err)
	assert.Equal(t, agent.PreviewOperationDelete, change.Operation)
	require.Len(t, change.Fields, 1)
	assert.Equal(t, "PRO-100", change.Fields[0].Before)
	assert.Nil(t, change.Fields[0].After)

	bare, err := Delete[testRecord](Record{Resource: testResource, Label: "Tile"}, nil)
	require.NoError(t, err)
	assert.Empty(t, bare.Fields)
	assert.Equal(t, "Tile", bare.Label)
}

func TestSend_CarriesTheRenderedMessage(t *testing.T) {
	t.Parallel()

	change := Send(Record{Resource: permission.ResourceCustomer, Label: "Acme"},
		&agent.MessagePreview{
			Channel: agent.MessageChannelEmail,
			To:      []string{"ap@acme.test", " "},
			Subject: "  Your delivery  ",
			Body:    "Hello",
		})

	assert.Equal(t, agent.PreviewOperationSend, change.Operation)
	require.NotNil(t, change.Message)
	assert.Equal(t, []string{"ap@acme.test"}, change.Message.To)
	assert.Equal(t, "Your delivery", change.Message.Subject)
}

func TestMoney_TotalsLinesAndHonoursSensitivity(t *testing.T) {
	t.Parallel()

	block := MoneyBlock("usd",
		agent.MoneyLine{
			Label:  "Linehaul",
			Before: decimal.NewNullDecimal(decimal.RequireFromString("1000")),
			After:  decimal.NewNullDecimal(decimal.RequireFromString("1200")),
		},
		agent.MoneyLine{
			Label: "Detention",
			After: decimal.NewNullDecimal(decimal.RequireFromString("150.25")),
		},
	)

	assert.Equal(t, "USD", block.Currency)
	assert.Equal(t, "1000", block.TotalBefore.Decimal.String())
	assert.Equal(t, "1350.25", block.TotalAfter.Decimal.String())
	assert.Equal(t, "350.25", block.Delta.Decimal.String())

	change := Money(Record{Resource: testResource}, block,
		SensitiveAs("ratingAmount"), WithRegistry(testRegistry(t)))
	require.NotNil(t, change.Money)
	assert.Equal(t, permission.SensitivityRestricted, change.Money.Sensitivity)

	confidential := Money(Record{Resource: testResource}, block,
		SensitiveAs("bankAccount"), WithRegistry(testRegistry(t)))
	assert.Nil(t, confidential.Money, "a confidential amount is never carried")
}

func TestBuild_BoundsWhatItKeeps(t *testing.T) {
	t.Parallel()

	changes := make([]*agent.RecordChange, 0, agent.MaxPreviewRecords+3)
	for range agent.MaxPreviewRecords + 3 {
		fields := make([]agent.PreviewFieldChange, 0, agent.MaxPreviewFieldsPerRecord+5)
		for j := range agent.MaxPreviewFieldsPerRecord + 5 {
			fields = append(fields, agent.PreviewFieldChange{
				Path:  "field" + string(rune('a'+j%26)),
				After: strings.Repeat("x", agent.MaxPreviewValueBytes+10),
			})
		}
		changes = append(changes, &agent.RecordChange{
			Operation: agent.PreviewOperationUpdate,
			Fields:    fields,
			Message: &agent.MessagePreview{
				Body: strings.Repeat("b", agent.MaxPreviewBodyBytes+1),
			},
		})
	}
	changes = append(changes, nil)

	preview := Build(" Would do a lot. ", changes...)

	assert.Equal(t, "Would do a lot.", preview.Summary)
	assert.Len(t, preview.Changes, agent.MaxPreviewRecords)
	assert.Equal(t, 3, preview.OmittedRecords)
	assert.True(t, preview.Partial)
	first := preview.Changes[0]
	assert.Len(t, first.Fields, agent.MaxPreviewFieldsPerRecord)
	assert.Equal(t, 5, first.OmittedFields)
	assert.True(t, first.Fields[0].Truncated)
	assert.LessOrEqual(t, len(first.Fields[0].After.(string)), agent.MaxPreviewValueBytes)
	assert.True(t, first.Message.BodyTruncated)
	assert.Len(t, first.Message.Body, agent.MaxPreviewBodyBytes)
}

func TestDescribe_IsTheParametersAndNothingMore(t *testing.T) {
	t.Parallel()

	preview := Describe("cancel_shipment", map[string]any{
		"shipmentId":   "shp_1",
		"cancelReason": "Customer pulled the load",
		"_owner":       "usr_1",
	})

	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "Would run cancel_shipment")
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, agent.PreviewOperationRun, preview.Changes[0].Operation)
	assert.Equal(t, []string{"cancelReason", "shipmentId"}, paths(&preview.Changes[0]),
		"the owner the runtime writes is not a parameter anyone set")
	assert.Equal(t, "Cancel reason", preview.Changes[0].Fields[0].Label)
}

func TestParameters_StampsEachParameterAndDropsConfidentialOnes(t *testing.T) {
	t.Parallel()

	preview := Parameters("update_record", testResource, map[string]any{
		"status":       "Hold",
		"bankAccount":  "000123",
		"ratingAmount": "12.50",
	}, WithRegistry(testRegistry(t)))

	require.Len(t, preview.Changes, 1)
	change := preview.Changes[0]
	assert.Equal(t, testResource, change.Resource)
	assert.Equal(t, []string{"ratingAmount", "status"}, paths(&change),
		"a confidential parameter is never shown")
	assert.Equal(t, permission.SensitivityRestricted, change.Fields[0].Sensitivity)
	assert.Equal(t, permission.SensitivityInternal, change.Fields[1].Sensitivity)
}

func TestFromSimulation_ReadsASimulationAsAPartialPreview(t *testing.T) {
	t.Parallel()

	record := pulid.MustNew("shp_")
	preview := FromSimulation(Record{Resource: testResource, ID: record}, &agent.ToolSimulation{
		Summary: "Would put PRO-100 on hold.",
		Changes: []agent.FieldChange{
			{Field: "status", From: "New", To: "Hold"},
			{Field: "bankAccount", To: "999"},
			{Field: " ", To: "ignored"},
		},
	}, WithRegistry(testRegistry(t)))

	assert.True(t, preview.Partial)
	assert.Equal(t, "Would put PRO-100 on hold.", preview.Summary)
	require.Len(t, preview.Changes, 1)
	change := preview.Changes[0]
	assert.Equal(t, record, change.EntityID)
	require.Len(t, change.Fields, 1)
	assert.Equal(t, "New", change.Fields[0].Before)
	assert.Equal(t, "Hold", change.Fields[0].After)
	assert.Nil(t, FromSimulation(Record{}, nil))
}

func TestChain_ProjectsALaterStepFromTheEarlierOne(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	other := pulid.MustNew("shp_")
	first := &agent.ProposalPreview{Changes: []agent.RecordChange{{
		Resource:  permission.ResourceShipment,
		EntityID:  shipment,
		Operation: agent.PreviewOperationUpdate,
		Fields:    []agent.PreviewFieldChange{{Path: "status", Before: "New", After: "Hold"}},
	}}}
	second := &agent.ProposalPreview{Changes: []agent.RecordChange{
		{
			Resource:  permission.ResourceShipment,
			EntityID:  shipment,
			Operation: agent.PreviewOperationUpdate,
			Fields: []agent.PreviewFieldChange{
				{Path: "status", Before: "New", After: "Cancelled"},
				{Path: "notes", Before: nil, After: "gone"},
			},
		},
		{
			Resource:  permission.ResourceShipment,
			EntityID:  other,
			Operation: agent.PreviewOperationUpdate,
			Fields:    []agent.PreviewFieldChange{{Path: "status", Before: "New", After: "Hold"}},
		},
	}}
	third := &agent.ProposalPreview{Changes: []agent.RecordChange{{
		Resource:  permission.ResourceShipment,
		EntityID:  shipment,
		Operation: agent.PreviewOperationUpdate,
		Fields:    []agent.PreviewFieldChange{{Path: "status", Before: "New", After: "New"}},
	}}}

	Chain(
		[]ChainStep{
			{Step: 1, Preview: first},
			{Step: 2, Preview: second},
			{Step: 3, Preview: third},
		},
	)

	assert.Zero(t, first.Changes[0].DependsOnStep)
	assert.Empty(t, first.Warnings)

	chained := second.Changes[0]
	assert.Equal(t, 1, chained.DependsOnStep)
	assert.Equal(t, "Hold", chained.Fields[0].Before, "starts from what step 1 leaves")
	assert.Equal(t, 1, chained.Fields[0].ProjectedFromStep)
	assert.Nil(t, chained.Fields[1].Before)
	assert.Zero(t, chained.Fields[1].ProjectedFromStep)
	assert.Zero(t, second.Changes[1].DependsOnStep, "another record is not chained")
	require.Len(t, second.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningDependsOnStep, second.Warnings[0].Code)
	assert.Equal(t, []string{"1"}, second.Warnings[0].Args)

	assert.Equal(t, 2, third.Changes[0].DependsOnStep, "the nearest earlier step")
	assert.Equal(t, "Cancelled", third.Changes[0].Fields[0].Before)
	assert.Equal(t, 2, third.Changes[0].Fields[0].ProjectedFromStep)
}
