package recordnumbers

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	shipmentKind = recordkind.Kind(permission.ResourceShipment)
	invoiceKind  = recordkind.Kind(permission.ResourceInvoice)
)

type fakeNumbers struct {
	numberings map[recordkind.Kind]recordnumberports.Numbering
	records    recordnumberports.NumberedRecords
	err        error
	asked      []map[recordkind.Kind][]string
}

func (f *fakeNumbers) Numbering(kind recordkind.Kind) (recordnumberports.Numbering, bool) {
	numbering, ok := f.numberings[kind]

	return numbering, ok
}

func (f *fakeNumbers) FindNumbers(
	_ context.Context,
	_ pagination.TenantInfo,
	numbers map[recordkind.Kind][]string,
) (recordnumberports.NumberedRecords, error) {
	f.asked = append(f.asked, numbers)

	return f.records, f.err
}

type rolePermissions struct {
	*agenttest.StubPermissions

	maxSensitivity permission.FieldSensitivity
}

func (p *rolePermissions) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		MaxSensitivity: p.maxSensitivity,
	}, nil
}

func newResolver(records *fakeNumbers, denied map[string]bool) *Resolver {
	return New(Params{
		Records: records,
		Permissions: &rolePermissions{
			StubPermissions: &agenttest.StubPermissions{Denied: denied},
			maxSensitivity:  permission.SensitivityInternal,
		},
	})
}

func defaultNumbers() *fakeNumbers {
	return &fakeNumbers{numberings: map[recordkind.Kind]recordnumberports.Numbering{
		shipmentKind: {Field: "proNumber", Name: "PRO number"},
		invoiceKind:  {Field: "number", Name: "invoice number"},
	}}
}

func contractFor(t *testing.T, schema, args map[string]any) argcheck.Contract {
	t.Helper()

	contract, err := argcheck.ContractCall(toolschema.NewValidator(), "stub", schema, args)
	require.NoError(t, err)

	return contract
}

func shipmentsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentIds": agenttoolschema.RecordIDs(permission.ResourceShipment,
				"The shipments, from shipment_list.", 25),
			"subjectId": agenttoolschema.KindID(
				"The record, from shipment_list or invoice_list.",
				shipmentKind, invoiceKind,
			),
			"customerId": agenttoolschema.RecordID(permission.ResourceCustomer,
				"The customer", "customer_list"),
		},
	}
}

func refusals(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	return argcheck.Problems(multiErr)
}

func TestResolve_ReadsEveryNumberOfAKindInOneRead(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	records := defaultNumbers()
	records.records = recordnumberports.NumberedRecords{shipmentKind: {
		"S26110493143683": {{ID: first, Number: "S26110493143683"}},
		"S26110579415132": {{ID: second, Number: "S26110579415132"}},
	}}

	resolved, err := newResolver(records, nil).Resolve(t.Context(), &Request{
		Actor:      runtimetest.Actor(),
		DataAccess: permission.SensitivityInternal,
		Contract: contractFor(t, shipmentsSchema(), map[string]any{
			"shipmentIds": []any{"S26110493143683", "S26110579415132"},
		}),
	})

	require.NoError(t, err)
	require.Len(t, records.asked, 1, "one read for the whole call")
	assert.Equal(t, map[recordkind.Kind][]string{
		shipmentKind: {"S26110493143683", "S26110579415132"},
	}, records.asked[0])
	assert.Equal(t, []any{first.String(), second.String()}, resolved.Args["shipmentIds"])
	assert.Contains(t, resolved.Note(), `shipmentIds[0] "S26110493143683" is the shipment `+
		`with that PRO number, id `+first.String())
}

func TestResolve_RefusesAsBeforeWhatCannotBeReadAsExactlyOneRecord(t *testing.T) {
	t.Parallel()

	shipment, invoice := pulid.MustNew("shp_"), pulid.MustNew("inv_")
	records := defaultNumbers()
	records.records = recordnumberports.NumberedRecords{
		shipmentKind: {"X-100": {{ID: shipment, Number: "X-100"}}},
		invoiceKind:  {"X-100": {{ID: invoice, Number: "X-100"}}},
	}

	_, err := newResolver(records, nil).Resolve(t.Context(), &Request{
		Actor: runtimetest.Actor(),
		Contract: contractFor(t, shipmentsSchema(), map[string]any{
			"shipmentIds": []any{"S404"},
			"subjectId":   "X-100",
			"customerId":  "ACME",
		}),
	})

	problems := refusals(t, err)
	require.Len(t, problems, 3)
	assert.Contains(t, problems[0], `customerId: "ACME" is not a record id`,
		"a kind with no number is refused as before")
	assert.Contains(t, problems[1], `shipmentIds[0]: "S404" is not a record id`,
		"a number no record carries is refused as before")
	assert.Contains(t, problems[2], "the shipment with PRO number X-100 or the invoice with "+
		"invoice number X-100")
	assert.Equal(t, map[recordkind.Kind][]string{
		shipmentKind: {"S404", "X-100"},
		invoiceKind:  {"X-100"},
	}, records.asked[0], "the customer, which has no number, is never read")
}

// The number is read as the tool's own read would be: a kind the person may
// not read, or whose number is above what the agent may see, is never read.
func TestResolve_ReadsOnlyKindsAndNumbersTheActorMaySee(t *testing.T) {
	t.Parallel()

	records := defaultNumbers()
	records.numberings[invoiceKind] = recordnumberports.Numbering{
		Field: "rerateVariancePercent",
		Name:  "invoice number",
	}

	_, err := newResolver(records, map[string]bool{
		permission.ResourceShipment.String() + ":" + string(permission.OpRead): true,
	}).Resolve(t.Context(), &Request{
		Actor:      runtimetest.Actor(),
		DataAccess: permission.SensitivityInternal,
		Contract: contractFor(t, shipmentsSchema(), map[string]any{
			"shipmentIds": []any{"S26110493143683"},
			"subjectId":   "INV-1",
		}),
	})

	assert.Len(t, refusals(t, err), 2)
	assert.Empty(t, records.asked, "nothing the actor may not see is looked up")
}

func TestResolve_AFailedReadIsNoRefusalOfTheModel(t *testing.T) {
	t.Parallel()

	records := defaultNumbers()
	records.err = errors.New("connection reset")

	_, err := newResolver(records, nil).Resolve(t.Context(), &Request{
		Actor: runtimetest.Actor(),
		Contract: contractFor(t, shipmentsSchema(), map[string]any{
			"shipmentIds": []any{"S26110493143683"},
		}),
	})

	require.Error(t, err)
	_, refused := errortypes.RefusalOf(err)
	assert.False(t, refused)

	result := argcheck.Refusal("stub", shipmentsSchema(), err)
	assert.Contains(t, result.Content, "its arguments could not be checked")
}
